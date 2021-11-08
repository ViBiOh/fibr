package exif

import (
	"context"
	"flag"
	"fmt"
	"net/http"

	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/httputils/v4/pkg/flags"
	"github.com/ViBiOh/httputils/v4/pkg/httpjson"
	"github.com/ViBiOh/httputils/v4/pkg/request"
	"github.com/prometheus/client_golang/prometheus"
)

const (
	exifMetadataFilename      = ""
	geocodeMetadataFilename   = "geocode"
	aggregateMetadataFilename = "aggregate"
)

var (
	metadataFilenames = []string{
		exifMetadataFilename,
		geocodeMetadataFilename,
		aggregateMetadataFilename,
	}

	exifDates = []string{
		"DateCreated",
		"CreateDate",
	}

	datePatterns = []string{
		"2006:01:02 15:04:05MST",
		"2006:01:02 15:04:05-07:00",
		"2006:01:02 15:04:05Z07:00",
		"2006:01:02 15:04:05",
		"2006:01:02",
		"01/02/2006 15:04:05",
		"1/02/2006 15:04:05",
	}
)

// App of package
type App struct {
	storageApp   provider.Storage
	done         chan struct{}
	geocodeQueue chan provider.StorageItem
	metrics      map[string]*prometheus.CounterVec

	geocodeURL  string
	exifRequest request.Request

	aggregateOnStart bool
	dateOnStart      bool
	directAccess     bool
	maxSize          int64
}

// Config of package
type Config struct {
	exifURL    *string
	exifUser   *string
	exifPass   *string
	geocodeURL *string

	maxSize          *int
	aggregateOnStart *bool
	dateOnStart      *bool
	directAccess     *bool
}

// Flags adds flags for configuring package
func Flags(fs *flag.FlagSet, prefix string, overrides ...flags.Override) Config {
	return Config{
		exifURL:  flags.New(prefix, "exif", "URL").Default("http://exas:1080", overrides).Label("Exif Tool URL (exas)").ToString(fs),
		exifUser: flags.New(prefix, "exif", "User").Default("", overrides).Label("Exif Tool URL Basic User").ToString(fs),
		exifPass: flags.New(prefix, "exif", "Password").Default("", overrides).Label("Exif Tool URL Basic Password").ToString(fs),

		geocodeURL: flags.New(prefix, "exif", "GeocodeURL").Default("", overrides).Label(fmt.Sprintf("Nominatim Geocode Service URL. This can leak GPS metadatas to a third-party (e.g. \"%s\")", publicNominatimURL)).ToString(fs),

		directAccess: flags.New(prefix, "exif", "DirectAccess").Default(false, nil).Label("Use Exas with direct access to filesystem (no large file upload to it, send a GET request)").ToBool(fs),
		maxSize:      flags.New(prefix, "exif", "MaxSize").Default(1024*1024*200, nil).Label("Max file size (in bytes) for extracting exif (0 to no limit)").ToInt(fs),

		dateOnStart:      flags.New(prefix, "exif", "DateOnStart").Default(false, nil).Label("Change file date from EXIF date on start").ToBool(fs),
		aggregateOnStart: flags.New(prefix, "exif", "AggregateOnStart").Default(false, nil).Label("Aggregate EXIF data per folder on start").ToBool(fs),
	}
}

// New creates new App from Config
func New(config Config, storageApp provider.Storage, prometheusRegisterer prometheus.Registerer) (App, error) {
	metrics, err := createMetrics(prometheusRegisterer, "exif", "geocode", "aggregate")
	if err != nil {
		return App{}, err
	}

	return App{
		exifRequest:      request.New().URL(*config.exifURL).BasicAuth(*config.exifUser, *config.exifPass),
		geocodeURL:       *config.geocodeURL,
		dateOnStart:      *config.dateOnStart,
		aggregateOnStart: *config.aggregateOnStart,
		directAccess:     *config.directAccess,
		maxSize:          int64(*config.maxSize),

		storageApp: storageApp,

		metrics:      metrics,
		done:         make(chan struct{}),
		geocodeQueue: make(chan provider.StorageItem, 10),
	}, nil
}

func (a App) enabled() bool {
	return !a.exifRequest.IsZero()
}

// Start worker
func (a App) Start(done <-chan struct{}) {
	defer close(a.geocodeQueue)
	defer close(a.done)

	if !a.enabled() {
		return
	}

	go a.processGeocodeQueue()

	<-done
}

func (a App) get(item provider.StorageItem) (map[string]interface{}, error) {
	exif, err := a.loadExif(item)
	if err != nil {
		return nil, fmt.Errorf("unable to load exif: %s", err)
	}

	if len(exif) != 0 {
		return exif, nil
	}

	exif, err = a.fetchAndStoreExif(item)
	if err != nil {
		return nil, fmt.Errorf("unable to fetch: %s", err)
	}

	return exif, nil
}

func (a App) fetchAndStoreExif(item provider.StorageItem) (map[string]interface{}, error) {
	a.increaseMetric("exif", "requested")

	resp, err := a.requestExas(context.Background(), item)
	if err != nil {
		return nil, fmt.Errorf("unable to request exif: %s", err)
	}

	var exifs []map[string]interface{}
	if err := httpjson.Read(resp, &exifs); err != nil {
		return nil, fmt.Errorf("unable to read exas response: %s", err)
	}

	var data map[string]interface{}
	if len(exifs) > 0 {
		data = exifs[0]
	}

	if err := a.saveMetadata(item, exifMetadataFilename, data); err != nil {
		return nil, fmt.Errorf("unable to save exif: %s", err)
	}

	return data, nil
}

func (a App) requestExas(ctx context.Context, item provider.StorageItem) (*http.Response, error) {
	if a.directAccess {
		return a.exifRequest.Method(http.MethodGet).Path(item.Pathname).Send(ctx, nil)
	}

	return provider.SendLargeFile(ctx, a.storageApp, item, a.exifRequest.Method(http.MethodPost))
}
