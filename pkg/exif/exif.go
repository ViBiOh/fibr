package exif

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/httputils/v4/pkg/flags"
	"github.com/ViBiOh/httputils/v4/pkg/httpjson"
	"github.com/ViBiOh/httputils/v4/pkg/request"
	"github.com/prometheus/client_golang/prometheus"
)

const (
	maxExifSize = 150 << 20 // 150mo

	exifMetadataFilename      = ""
	geocodeMetadataFilename   = "geocode"
	aggregateMetadataFilename = "aggregate"
)

var (
	exasClient = &http.Client{
		Timeout: 2 * time.Minute,
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

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
	storageApp provider.Storage

	done         chan struct{}
	geocodeQueue chan provider.StorageItem

	metrics map[string]*prometheus.CounterVec

	exifURL          string
	geocodeURL       string
	dateOnStart      bool
	aggregateOnStart bool
}

// Config of package
type Config struct {
	exifURL          *string
	geocodeURL       *string
	dateOnStart      *bool
	aggregateOnStart *bool
}

// Flags adds flags for configuring package
func Flags(fs *flag.FlagSet, prefix string, overrides ...flags.Override) Config {
	return Config{
		exifURL:          flags.New(prefix, "exif", "URL").Default("http://exas:1080", overrides).Label("Exif Tool URL (exas)").ToString(fs),
		geocodeURL:       flags.New(prefix, "exif", "GeocodeURL").Default("", overrides).Label(fmt.Sprintf("Nominatim Geocode Service URL. This can leak GPS metadatas to a third-party (e.g. \"%s\")", publicNominatimURL)).ToString(fs),
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
		exifURL:          strings.TrimSpace(*config.exifURL),
		geocodeURL:       strings.TrimSpace(*config.geocodeURL),
		dateOnStart:      *config.dateOnStart,
		aggregateOnStart: *config.aggregateOnStart,

		storageApp: storageApp,

		metrics:      metrics,
		done:         make(chan struct{}),
		geocodeQueue: make(chan provider.StorageItem, 10),
	}, nil
}

func (a App) enabled() bool {
	return len(a.exifURL) != 0
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
	file, err := a.storageApp.ReaderFrom(item.Pathname) // file will be closed by `.Send`
	if err != nil {
		return nil, fmt.Errorf("unable to get reader: %s", err)
	}

	a.increaseMetric("exif", "requested")

	resp, err := request.New().WithClient(exasClient).Post(a.exifURL).Send(context.Background(), file)
	if err != nil {
		return nil, err
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
