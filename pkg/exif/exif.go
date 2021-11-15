package exif

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"strings"

	"github.com/ViBiOh/exas/pkg/model"
	"github.com/ViBiOh/fibr/pkg/provider"
	amqpclient "github.com/ViBiOh/httputils/v4/pkg/amqp"
	"github.com/ViBiOh/httputils/v4/pkg/flags"
	"github.com/ViBiOh/httputils/v4/pkg/httpjson"
	prom "github.com/ViBiOh/httputils/v4/pkg/prometheus"
	"github.com/ViBiOh/httputils/v4/pkg/request"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/streadway/amqp"
)

const (
	exifMetadataFilename      = ""
	aggregateMetadataFilename = "aggregate"
)

var metadataFilenames = []string{
	exifMetadataFilename,
	aggregateMetadataFilename,
}

// App of package
type App struct {
	storageApp      provider.Storage
	exifMetric      *prometheus.CounterVec
	aggregateMetric *prometheus.CounterVec

	exifRequest request.Request

	amqpClient     *amqpclient.Client
	amqpExchange   string
	amqpRoutingKey string

	directAccess bool
	maxSize      int64
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
func Flags(fs *flag.FlagSet, prefix string, overrides ...flags.Override) Config {
	return Config{
		exifURL:  flags.New(prefix, "exif", "URL").Default("http://exas:1080", overrides).Label("Exif Tool URL (exas)").ToString(fs),
		exifUser: flags.New(prefix, "exif", "User").Default("", overrides).Label("Exif Tool URL Basic User").ToString(fs),
		exifPass: flags.New(prefix, "exif", "Password").Default("", overrides).Label("Exif Tool URL Basic Password").ToString(fs),

		directAccess: flags.New(prefix, "exif", "DirectAccess").Default(false, nil).Label("Use Exas with direct access to filesystem (no large file upload, send a GET request, Basic Auth recommended)").ToBool(fs),
		maxSize:      flags.New(prefix, "exif", "MaxSize").Default(1024*1024*200, nil).Label("Max file size (in bytes) for extracting exif (0 to no limit). Not used if DirectAccess enabled.").ToInt(fs),

		amqpExchange:   flags.New(prefix, "exif", "AmqpExchange").Default("fibr", nil).Label("AMQP Exchange Name").ToString(fs),
		amqpRoutingKey: flags.New(prefix, "exif", "AmqpRoutingKey").Default("exif", nil).Label("AMQP Routing Key for exif").ToString(fs),
	}
}

// New creates new App from Config
func New(config Config, storageApp provider.Storage, prometheusRegisterer prometheus.Registerer, amqpClient *amqpclient.Client) (App, error) {
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

		amqpExchange:   amqpExchange,
		amqpRoutingKey: strings.TrimSpace(*config.amqpRoutingKey),

		storageApp: storageApp,

		exifMetric:      prom.CounterVec(prometheusRegisterer, "fibr", "exif", "item", "state"),
		aggregateMetric: prom.CounterVec(prometheusRegisterer, "fibr", "aggregate", "item", "state"),
	}, nil
}

func (a App) enabled() bool {
	return !a.exifRequest.IsZero()
}

func (a App) get(item provider.StorageItem) (model.Exif, error) {
	exif, err := a.loadExif(item)
	if err != nil {
		return exif, fmt.Errorf("unable to load exif: %s", err)
	}

	if !exif.IsZero() || a.amqpClient != nil {
		return exif, nil
	}

	data, err := a.extractExif(context.Background(), item)
	if err != nil {
		return exif, fmt.Errorf("unable to extract exif: %s", err)
	}

	if err := a.saveMetadata(item, exifMetadataFilename, data); err != nil {
		return exif, fmt.Errorf("unable to save exif: %s", err)
	}

	return a.loadExif(item)
}

func (a App) extractExif(ctx context.Context, item provider.StorageItem) (map[string]interface{}, error) {
	var data map[string]interface{}
	var resp *http.Response
	var err error

	if a.directAccess {
		resp, err = a.exifRequest.Method(http.MethodGet).Path(item.Pathname).Send(ctx, nil)
	} else {
		resp, err = provider.SendLargeFile(ctx, a.storageApp, item, a.exifRequest.Method(http.MethodPost))
	}

	if err != nil {
		return data, fmt.Errorf("unable to fetch exif: %s", err)
	}

	if err := httpjson.Read(resp, &data); err != nil {
		return data, fmt.Errorf("unable to read exif: %s", err)
	}

	return data, nil
}

func (a App) askForExif(item provider.StorageItem) error {
	payload, err := json.Marshal(item)
	if err != nil {
		return fmt.Errorf("unable to marshal stream amqp message: %s", err)
	}

	if err = a.amqpClient.Publish(amqp.Publishing{
		ContentType: "application/json",
		Body:        payload,
	}, a.amqpExchange, a.amqpRoutingKey); err != nil {
		return fmt.Errorf("unable to publish exif amqp message: %s", err)
	}

	return nil
}
