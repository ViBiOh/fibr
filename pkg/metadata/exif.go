package metadata

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"strings"

	absto "github.com/ViBiOh/absto/pkg/model"
	exas "github.com/ViBiOh/exas/pkg/model"
	"github.com/ViBiOh/fibr/pkg/exclusive"
	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/flags"
	amqpclient "github.com/ViBiOh/httputils/v4/pkg/amqp"
	"github.com/ViBiOh/httputils/v4/pkg/cache"
	"github.com/ViBiOh/httputils/v4/pkg/httpjson"
	prom "github.com/ViBiOh/httputils/v4/pkg/prometheus"
	"github.com/ViBiOh/httputils/v4/pkg/redis"
	"github.com/ViBiOh/httputils/v4/pkg/request"
	"github.com/ViBiOh/httputils/v4/pkg/tracer"
	"github.com/prometheus/client_golang/prometheus"
	"go.opentelemetry.io/otel/trace"
)

var errInvalidItemType = errors.New("invalid item type")

type App struct {
	tracer            trace.Tracer
	storageApp        absto.Storage
	listStorageApp    absto.Storage
	exifMetric        *prometheus.CounterVec
	aggregateMetric   *prometheus.CounterVec
	exifCacheApp      *cache.App[absto.Item, provider.Metadata]
	aggregateCacheApp *cache.App[absto.Item, provider.Aggregate]

	exclusiveApp exclusive.App
	redisClient  redis.Client

	amqpClient     *amqpclient.Client
	amqpExchange   string
	amqpRoutingKey string

	exifRequest request.Request

	maxSize      int64
	directAccess bool
}

type Config struct {
	exifURL  *string
	exifUser *string
	exifPass *string

	amqpExchange   *string
	amqpRoutingKey *string

	maxSize      *int
	directAccess *bool
}

func Flags(fs *flag.FlagSet, prefix string) Config {
	return Config{
		exifURL:  flags.New("URL", "Exif Tool URL (exas)").Prefix(prefix).DocPrefix("exif").String(fs, "http://exas:1080", nil),
		exifUser: flags.New("User", "Exif Tool URL Basic User").Prefix(prefix).DocPrefix("exif").String(fs, "", nil),
		exifPass: flags.New("Password", "Exif Tool URL Basic Password").Prefix(prefix).DocPrefix("exif").String(fs, "", nil),

		directAccess: flags.New("DirectAccess", "Use Exas with direct access to filesystem (no large file upload, send a GET request, Basic Auth recommended)").Prefix(prefix).DocPrefix("exif").Bool(fs, false, nil),
		maxSize:      flags.New("MaxSize", "Max file size (in bytes) for extracting exif (0 to no limit). Not used if DirectAccess enabled.").Prefix(prefix).DocPrefix("exif").Int(fs, 1024*1024*200, nil),

		amqpExchange:   flags.New("AmqpExchange", "AMQP Exchange Name").Prefix(prefix).DocPrefix("exif").String(fs, "fibr", nil),
		amqpRoutingKey: flags.New("AmqpRoutingKey", "AMQP Routing Key for exif").Prefix(prefix).DocPrefix("exif").String(fs, "exif_input", nil),
	}
}

func New(config Config, storageApp absto.Storage, prometheusRegisterer prometheus.Registerer, tracerApp tracer.App, amqpClient *amqpclient.Client, redisClient redis.Client, exclusiveApp exclusive.App) (App, error) {
	var amqpExchange string

	if amqpClient != nil {
		amqpExchange = strings.TrimSpace(*config.amqpExchange)

		if err := amqpClient.Publisher(amqpExchange, "direct", nil); err != nil {
			return App{}, fmt.Errorf("configure amqp: %w", err)
		}
	}

	app := App{
		exifRequest:  request.New().URL(strings.TrimSpace(*config.exifURL)).BasicAuth(strings.TrimSpace(*config.exifUser), *config.exifPass),
		directAccess: *config.directAccess,
		maxSize:      int64(*config.maxSize),

		redisClient: redisClient,

		amqpClient:     amqpClient,
		amqpExchange:   amqpExchange,
		amqpRoutingKey: strings.TrimSpace(*config.amqpRoutingKey),

		tracer:       tracerApp.GetTracer("exif"),
		exclusiveApp: exclusiveApp,
		storageApp:   storageApp,
		listStorageApp: storageApp.WithIgnoreFn(func(item absto.Item) bool {
			return !strings.HasSuffix(item.Name(), ".json")
		}),

		exifMetric:      prom.CounterVec(prometheusRegisterer, "fibr", "exif", "item", "state"),
		aggregateMetric: prom.CounterVec(prometheusRegisterer, "fibr", "aggregate", "item", "state"),
	}

	app.exifCacheApp = cache.New(redisClient, redisKey, func(ctx context.Context, item absto.Item) (provider.Metadata, error) {
		if item.IsDir() {
			return provider.Metadata{}, errInvalidItemType
		}

		return app.loadExif(ctx, item)
	}, cacheDuration, provider.MaxConcurrency, tracerApp.GetTracer("exif_cache"))

	app.aggregateCacheApp = cache.New(redisClient, redisKey, func(ctx context.Context, item absto.Item) (provider.Aggregate, error) {
		if !item.IsDir() {
			return provider.Aggregate{}, errInvalidItemType
		}

		return app.loadAggregate(ctx, item)
	}, cacheDuration, provider.MaxConcurrency, tracerApp.GetTracer("aggregate_cache"))

	return app, nil
}

func (a App) ListDir(ctx context.Context, item absto.Item) ([]absto.Item, error) {
	if !item.IsDir() {
		return nil, nil
	}

	exifs, err := a.listStorageApp.List(ctx, provider.MetadataDirectory(item))
	if err != nil && !absto.IsNotExist(err) {
		return exifs, err
	}
	return exifs, nil
}

func (a App) enabled() bool {
	return !a.exifRequest.IsZero()
}

func (a App) extractAndSaveExif(ctx context.Context, item absto.Item) (provider.Metadata, error) {
	exif, err := a.extractExif(ctx, item)
	if err != nil {
		return provider.Metadata{}, fmt.Errorf("extract exif: %w", err)
	}

	return a.Update(ctx, item, provider.ReplaceExif(exif))
}

func (a App) extractExif(ctx context.Context, item absto.Item) (exif exas.Exif, err error) {
	var resp *http.Response

	a.increaseExif("request")

	if a.directAccess {
		resp, err = a.exifRequest.Method(http.MethodGet).Path(item.Pathname).Send(ctx, nil)
	} else {
		resp, err = provider.SendLargeFile(ctx, a.storageApp, item, a.exifRequest.Method(http.MethodPost))
	}

	if err != nil {
		a.increaseExif("error")
		err = fmt.Errorf("fetch exif: %w", err)
		return
	}

	if err = httpjson.Read(resp, &exif); err != nil {
		err = fmt.Errorf("read exif: %w", err)
	}

	return
}

func (a App) publishExifRequest(ctx context.Context, item absto.Item) error {
	a.increaseExif("publish")

	return a.amqpClient.PublishJSON(ctx, item, a.amqpExchange, a.amqpRoutingKey)
}
