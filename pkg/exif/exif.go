package exif

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"strings"

	absto "github.com/ViBiOh/absto/pkg/model"
	exas "github.com/ViBiOh/exas/pkg/model"
	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/flags"
	amqpclient "github.com/ViBiOh/httputils/v4/pkg/amqp"
	"github.com/ViBiOh/httputils/v4/pkg/httpjson"
	"github.com/ViBiOh/httputils/v4/pkg/logger"
	prom "github.com/ViBiOh/httputils/v4/pkg/prometheus"
	"github.com/ViBiOh/httputils/v4/pkg/request"
	"github.com/ViBiOh/httputils/v4/pkg/tracer"
	"github.com/prometheus/client_golang/prometheus"
	"go.opentelemetry.io/otel/trace"
)

// App of package
type App struct {
	tracer          trace.Tracer
	storageApp      absto.Storage
	listStorageApp  absto.Storage
	exifMetric      *prometheus.CounterVec
	aggregateMetric *prometheus.CounterVec

	amqpClient     *amqpclient.Client
	amqpExchange   string
	amqpRoutingKey string

	exifRequest request.Request

	maxSize      int64
	directAccess bool
}

// Config of package
type Config struct {
	exifURL  *string
	exifUser *string
	exifPass *string

	amqpExchange   *string
	amqpRoutingKey *string

	maxSize      *int
	directAccess *bool
}

// Flags adds flags for configuring package
func Flags(fs *flag.FlagSet, prefix string) Config {
	return Config{
		exifURL:  flags.String(fs, prefix, "exif", "URL", "Exif Tool URL (exas)", "http://exas:1080", nil),
		exifUser: flags.String(fs, prefix, "exif", "User", "Exif Tool URL Basic User", "", nil),
		exifPass: flags.String(fs, prefix, "exif", "Password", "Exif Tool URL Basic Password", "", nil),

		directAccess: flags.Bool(fs, prefix, "exif", "DirectAccess", "Use Exas with direct access to filesystem (no large file upload, send a GET request, Basic Auth recommended)", false, nil),
		maxSize:      flags.Int(fs, prefix, "exif", "MaxSize", "Max file size (in bytes) for extracting exif (0 to no limit). Not used if DirectAccess enabled.", 1024*1024*200, nil),

		amqpExchange:   flags.String(fs, prefix, "exif", "AmqpExchange", "AMQP Exchange Name", "fibr", nil),
		amqpRoutingKey: flags.String(fs, prefix, "exif", "AmqpRoutingKey", "AMQP Routing Key for exif", "exif_input", nil),
	}
}

// New creates new App from Config
func New(config Config, storageApp absto.Storage, prometheusRegisterer prometheus.Registerer, tracerApp tracer.App, amqpClient *amqpclient.Client) (App, error) {
	var amqpExchange string
	if amqpClient != nil {
		amqpExchange = strings.TrimSpace(*config.amqpExchange)

		if err := amqpClient.Publisher(amqpExchange, "direct", nil); err != nil {
			return App{}, fmt.Errorf("unable to configure amqp: %s", err)
		}
	}

	return App{
		exifRequest:  request.New().URL(strings.TrimSpace(*config.exifURL)).BasicAuth(strings.TrimSpace(*config.exifUser), *config.exifPass),
		directAccess: *config.directAccess,
		maxSize:      int64(*config.maxSize),

		amqpClient:     amqpClient,
		amqpExchange:   amqpExchange,
		amqpRoutingKey: strings.TrimSpace(*config.amqpRoutingKey),

		tracer:     tracerApp.GetTracer("exif"),
		storageApp: storageApp,
		listStorageApp: storageApp.WithIgnoreFn(func(item absto.Item) bool {
			return !strings.HasSuffix(item.Name, ".json")
		}),

		exifMetric:      prom.CounterVec(prometheusRegisterer, "fibr", "exif", "item", "state"),
		aggregateMetric: prom.CounterVec(prometheusRegisterer, "fibr", "aggregate", "item", "state"),
	}, nil
}

// ListDir return all exifs for a given directory
func (a App) ListDir(ctx context.Context, item absto.Item) ([]absto.Item, error) {
	if !item.IsDir {
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

func (a App) extractAndSaveExif(ctx context.Context, item absto.Item) (exif exas.Exif, err error) {
	exif, err = a.extractExif(ctx, item)
	if err != nil {
		err = fmt.Errorf("unable to extract exif: %s", err)
		return
	}

	previousExif, err := a.loadExif(ctx, item)
	if err != nil && !absto.IsNotExist(err) {
		logger.WithField("item", item.Pathname).Error("unable to load exif: %s", err)
	}

	exif.Description = previousExif.Description

	if exif.IsZero() {
		return
	}

	if err = a.saveMetadata(ctx, item, exif); err != nil {
		err = fmt.Errorf("unable to save exif: %s", err)
	}

	return
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
		err = fmt.Errorf("unable to fetch exif: %s", err)
		return
	}

	if err = httpjson.Read(resp, &exif); err != nil {
		err = fmt.Errorf("unable to read exif: %s", err)
	}

	return
}

func (a App) publishExifRequest(item absto.Item) error {
	a.increaseExif("publish")

	return a.amqpClient.PublishJSON(item, a.amqpExchange, a.amqpRoutingKey)
}
