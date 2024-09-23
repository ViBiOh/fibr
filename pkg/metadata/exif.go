package metadata

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"unique"

	absto "github.com/ViBiOh/absto/pkg/model"
	exas "github.com/ViBiOh/exas/pkg/model"
	"github.com/ViBiOh/fibr/pkg/exclusive"
	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/flags"
	amqpclient "github.com/ViBiOh/httputils/v4/pkg/amqp"
	"github.com/ViBiOh/httputils/v4/pkg/cache"
	"github.com/ViBiOh/httputils/v4/pkg/httpjson"
	"github.com/ViBiOh/httputils/v4/pkg/redis"
	"github.com/ViBiOh/httputils/v4/pkg/request"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

var errInvalidItemType = errors.New("invalid item type")

type Service struct {
	tracer          trace.Tracer
	storage         absto.Storage
	listStorage     absto.Storage
	exifMetric      metric.Int64Counter
	aggregateMetric metric.Int64Counter
	exifCache       *cache.Cache[absto.Item, provider.Metadata]
	aggregateCache  *cache.Cache[absto.Item, provider.Aggregate]

	exclusive   exclusive.Service
	redisClient redis.Client

	amqpClient     *amqpclient.Client
	amqpExchange   string
	amqpRoutingKey string

	exifRequest request.Request

	maxSize      int64
	directAccess bool
}

type Config struct {
	ExifURL  string
	ExifUser string
	ExifPass string

	AmqpExchange   string
	AmqpRoutingKey string

	MaxSize      int64
	DirectAccess bool
}

func Flags(fs *flag.FlagSet, prefix string) *Config {
	var config Config

	flags.New("URL", "Exif Tool URL (exas)").Prefix(prefix).DocPrefix("exif").StringVar(fs, &config.ExifURL, "http://exas:1080", nil)
	flags.New("User", "Exif Tool URL Basic User").Prefix(prefix).DocPrefix("exif").StringVar(fs, &config.ExifUser, "", nil)
	flags.New("Password", "Exif Tool URL Basic Password").Prefix(prefix).DocPrefix("exif").StringVar(fs, &config.ExifPass, "", nil)

	flags.New("DirectAccess", "Use Exas with direct access to filesystem (no large file upload, send a GET request, Basic Auth recommended)").Prefix(prefix).DocPrefix("exif").BoolVar(fs, &config.DirectAccess, false, nil)
	flags.New("MaxSize", "Max file size (in bytes) for extracting exif (0 to no limit). Not used if DirectAccess enabled.").Prefix(prefix).DocPrefix("exif").Int64Var(fs, &config.MaxSize, 1024*1024*200, nil)

	flags.New("AmqpExchange", "AMQP Exchange Name").Prefix(prefix).DocPrefix("exif").StringVar(fs, &config.AmqpExchange, "fibr", nil)
	flags.New("AmqpRoutingKey", "AMQP Routing Key for exif").Prefix(prefix).DocPrefix("exif").StringVar(fs, &config.AmqpRoutingKey, "exif_input", nil)

	return &config
}

func New(ctx context.Context, config *Config, storageService absto.Storage, meterProvider metric.MeterProvider, traceProvider trace.TracerProvider, amqpClient *amqpclient.Client, redisClient redis.Client, exclusiveService exclusive.Service) (Service, error) {
	var amqpExchange string

	if amqpClient != nil {
		amqpExchange = config.AmqpExchange

		if err := amqpClient.Publisher(amqpExchange, "direct", nil); err != nil {
			return Service{}, fmt.Errorf("configure amqp: %w", err)
		}
	}

	service := Service{
		exifRequest:  request.New().URL(config.ExifURL).BasicAuth(config.ExifUser, config.ExifPass),
		directAccess: config.DirectAccess,
		maxSize:      config.MaxSize,

		redisClient: redisClient,

		amqpClient:     amqpClient,
		amqpExchange:   amqpExchange,
		amqpRoutingKey: config.AmqpRoutingKey,

		tracer:    traceProvider.Tracer("exif"),
		exclusive: exclusiveService,
		storage:   storageService,
		listStorage: storageService.WithIgnoreFn(func(item absto.Item) bool {
			return !strings.HasSuffix(item.Name(), ".json")
		}),
	}

	if meterProvider != nil {
		meter := meterProvider.Meter("github.com/ViBiOh/fibr/pkg/metadata/exif")

		var err error

		service.exifMetric, err = meter.Int64Counter("fibr.exif")
		if err != nil {
			slog.LogAttrs(ctx, slog.LevelError, "create exif counter", slog.Any("error", err))
		}

		service.aggregateMetric, err = meter.Int64Counter("fibr.aggregate")
		if err != nil {
			slog.LogAttrs(ctx, slog.LevelError, "create aggregate counter", slog.Any("error", err))
		}
	}

	service.exifCache = cache.New(redisClient, redisKey, func(ctx context.Context, item absto.Item) (provider.Metadata, error) {
		if item.IsDir() {
			return provider.Metadata{}, errInvalidItemType
		}

		metadata, err := service.loadExif(ctx, item)

		for index, tag := range metadata.Tags {
			metadata.Tags[index] = unique.Make(tag).Value()
		}

		for key, value := range metadata.Exif.Data {
			switch value.(type) {
			case string:
				metadata.Exif.Data[unique.Make(key).Value()] = unique.Make(value).Value()
			}
		}

		return metadata, err
	}, traceProvider).
		WithMaxConcurrency(provider.MaxConcurrency).
		WithClientSideCaching(ctx, "fibr_exif", provider.MaxClientSideCaching)

	service.aggregateCache = cache.New(redisClient, redisKey, func(ctx context.Context, item absto.Item) (provider.Aggregate, error) {
		if !item.IsDir() {
			return provider.Aggregate{}, errInvalidItemType
		}

		return service.loadAggregate(ctx, item)
	}, traceProvider).
		WithMaxConcurrency(provider.MaxConcurrency).
		WithClientSideCaching(ctx, "fibr_aggregate", provider.MaxClientSideCaching)

	return service, nil
}

func (s Service) ListDir(ctx context.Context, item absto.Item) ([]absto.Item, error) {
	if !item.IsDir() {
		return nil, nil
	}

	exifs, err := s.listStorage.List(ctx, provider.MetadataDirectory(item))
	if err != nil && !absto.IsNotExist(err) {
		return exifs, err
	}
	return exifs, nil
}

func (s Service) enabled() bool {
	return !s.exifRequest.IsZero()
}

func (s Service) extractAndSaveExif(ctx context.Context, item absto.Item) (provider.Metadata, error) {
	exif, err := s.extractExif(ctx, item)
	if err != nil {
		return provider.Metadata{}, fmt.Errorf("extract exif: %w", err)
	}

	return s.Update(ctx, item, provider.ReplaceExif(exif))
}

func (s Service) extractExif(ctx context.Context, item absto.Item) (exif exas.Exif, err error) {
	var resp *http.Response

	s.increaseExif(ctx, "request")

	if s.directAccess {
		resp, err = s.exifRequest.Method(http.MethodGet).Path(item.Pathname).Send(ctx, nil)
	} else {
		resp, err = provider.SendLargeFile(ctx, s.storage, item, s.exifRequest.Method(http.MethodPost))
	}

	if err != nil {
		s.increaseExif(ctx, "error")
		err = fmt.Errorf("fetch exif: %w", err)
		return
	}

	exif, err = httpjson.Read[exas.Exif](resp)
	if err != nil {
		err = fmt.Errorf("read exif: %w", err)
	}

	return
}

func (s Service) publishExifRequest(ctx context.Context, item absto.Item) error {
	s.increaseExif(ctx, "publish")

	return s.amqpClient.PublishJSON(ctx, item, s.amqpExchange, s.amqpRoutingKey)
}
