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
	"github.com/ViBiOh/httputils/v4/pkg/logger"
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
	exasClient = http.Client{
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
type App interface {
	Start(<-chan struct{})
	HasExif(provider.StorageItem) bool
	HasGeocode(provider.StorageItem) bool
	HasAggregat(provider.StorageItem) bool
	ExtractFor(provider.StorageItem)
	ExtractGeocodeFor(provider.StorageItem)
	UpdateDateFor(provider.StorageItem)
	AggregateFor(provider.StorageItem)
	GetAggregateFor(provider.StorageItem) (provider.Aggregate, error)
	Rename(provider.StorageItem, provider.StorageItem)
	Delete(provider.StorageItem)
}

// Config of package
type Config struct {
	exifURL    *string
	geocodeURL *string
}

type app struct {
	storageApp provider.Storage

	done           chan struct{}
	geocodeQueue   chan provider.StorageItem
	aggregateQueue chan provider.StorageItem

	metrics map[string]*prometheus.GaugeVec

	exifURL    string
	geocodeURL string
}

// Flags adds flags for configuring package
func Flags(fs *flag.FlagSet, prefix string, overrides ...flags.Override) Config {
	return Config{
		exifURL:    flags.New(prefix, "exif").Name("URL").Default(flags.Default("URL", "http://exas:1080", overrides)).Label("Exif Tool URL (exas)").ToString(fs),
		geocodeURL: flags.New(prefix, "exif").Name("GeocodeURL").Default(flags.Default("URL", "", overrides)).Label(fmt.Sprintf("Nominatim Geocode Service URL. This can leak GPS metadatas to a third-party (e.g. \"%s\")", publicNominatimURL)).ToString(fs),
	}
}

// New creates new App from Config
func New(config Config, storageApp provider.Storage, prometheusRegisterer prometheus.Registerer) App {
	return app{
		exifURL:    strings.TrimSpace(*config.exifURL),
		geocodeURL: strings.TrimSpace(*config.geocodeURL),

		storageApp: storageApp,

		metrics:        createMetrics(prometheusRegisterer),
		done:           make(chan struct{}),
		geocodeQueue:   make(chan provider.StorageItem, 10),
		aggregateQueue: make(chan provider.StorageItem, 10),
	}
}

func (a app) enabled() bool {
	return len(a.exifURL) != 0
}

func (a app) Start(done <-chan struct{}) {
	defer close(a.done)
	defer close(a.geocodeQueue)
	defer close(a.aggregateQueue)

	if !a.enabled() {
		return
	}

	go a.processGeocodeQueue()
	go a.processAggregateQueue()

	<-done
}

func (a app) ExtractFor(item provider.StorageItem) {
	if !a.enabled() {
		return
	}

	if item.IsDir {
		return
	}

	if _, err := a.fetchAndStoreExif(item); err != nil {
		logger.Error("unable to fetch and store for `%s`: %s", item.Pathname, err)
	}
}

func (a app) get(item provider.StorageItem) (map[string]interface{}, error) {
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

func (a app) fetchAndStoreExif(item provider.StorageItem) (map[string]interface{}, error) {
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

	a.increaseMetric("exif", "saved")

	return data, nil
}

func (a app) Rename(old, new provider.StorageItem) {
	if !a.enabled() {
		return
	}

	for _, suffix := range metadataFilenames {
		oldPath := getExifPath(old, suffix)
		if _, err := a.storageApp.Info(oldPath); provider.IsNotExist(err) {
			return
		}

		if err := a.storageApp.Rename(oldPath, getExifPath(new, suffix)); err != nil {
			logger.Error("unable to rename exif: %s", err)
		}
	}

	if !old.IsDir {
		oldDir, err := a.getDirOf(old)
		if err != nil {
			logger.Error("unable to get directory for `%s`: %s", old.Pathname, err)
		}

		newDir, err := a.getDirOf(new)
		if err != nil {
			logger.Error("unable to get directory for `%s`: %s", old.Pathname, err)
		}

		if oldDir.Pathname != newDir.Pathname {
			a.AggregateFor(oldDir)
			a.AggregateFor(newDir)
		}
	}
}

func (a app) Delete(item provider.StorageItem) {
	if !a.enabled() {
		return
	}

	for _, suffix := range metadataFilenames {
		if err := a.storageApp.Remove(getExifPath(item, suffix)); err != nil {
			logger.Error("unable to delete exif: %s", err)
		}
	}

	if !item.IsDir {
		a.AggregateFor(item)
	}
}
