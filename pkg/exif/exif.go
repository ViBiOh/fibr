package exif

import (
	"context"
	"encoding/json"
	"errors"
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
)

var (
	exasClient = http.Client{
		Timeout: 2 * time.Minute,
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	exifSuffixes = []string{"", "geocode"}

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
	ExtractFor(provider.StorageItem)
	ExtractGeocodeFor(provider.StorageItem)
	UpdateDate(provider.StorageItem)
	GeolocationFor(provider.StorageItem) (string, error)
	Rename(provider.StorageItem, provider.StorageItem)
	Delete(provider.StorageItem)
}

// Config of package
type Config struct {
	exifURL    *string
	geocodeURL *string
}

type app struct {
	storageApp   provider.Storage
	geocodeDone  chan struct{}
	geocodeQueue chan provider.StorageItem

	exifCounter    *prometheus.GaugeVec
	dateCounter    *prometheus.GaugeVec
	geocodeCounter *prometheus.GaugeVec

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
	exifCounter := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "fibr",
		Subsystem: "exif",
		Name:      "item",
	}, []string{"state"})
	dateCounter := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "fibr",
		Subsystem: "date",
		Name:      "item",
	}, []string{"state"})
	geocodeCounter := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "fibr",
		Subsystem: "geocode",
		Name:      "item",
	}, []string{"state"})

	if prometheusRegisterer != nil {
		if err := prometheusRegisterer.Register(exifCounter); err != nil {
			logger.Error("unable to register exif gauge: %s", err)
		}
		if err := prometheusRegisterer.Register(dateCounter); err != nil {
			logger.Error("unable to register date gauge: %s", err)
		}
		if err := prometheusRegisterer.Register(geocodeCounter); err != nil {
			logger.Error("unable to register geocode gauge: %s", err)
		}
	}

	return app{
		exifURL:    strings.TrimSpace(*config.exifURL),
		geocodeURL: strings.TrimSpace(*config.geocodeURL),

		storageApp: storageApp,

		exifCounter:    exifCounter,
		dateCounter:    dateCounter,
		geocodeCounter: geocodeCounter,
		geocodeDone:    make(chan struct{}),
		geocodeQueue:   make(chan provider.StorageItem, 10),
	}
}

func (a app) enabled() bool {
	return len(a.exifURL) != 0
}

func (a app) Start(done <-chan struct{}) {
	defer close(a.geocodeDone)
	defer close(a.geocodeQueue)

	if !a.enabled() {
		return
	}

	go a.fetchGeocodes()

	<-done
}

// CanHaveExif determine if exif can be extracted for given pathname
func CanHaveExif(item provider.StorageItem) bool {
	return (item.IsImage() || item.IsVideo() || item.IsPdf()) && item.Size < maxExifSize
}

func (a app) HasExif(item provider.StorageItem) bool {
	return a.hasMetadata(item, "")
}

func (a app) HasGeocode(item provider.StorageItem) bool {
	return a.hasMetadata(item, "geocode")
}

func (a app) hasMetadata(item provider.StorageItem, suffix string) bool {
	if !a.enabled() {
		return false
	}

	if item.IsDir {
		return false
	}

	_, err := a.storageApp.Info(getExifPath(item, suffix))
	return err == nil
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
	var data map[string]interface{}

	reader, err := a.storageApp.ReaderFrom(getExifPath(item, ""))
	if err == nil {
		if err := json.NewDecoder(reader).Decode(&data); err != nil {
			return nil, fmt.Errorf("unable to decode: %s", err)
		}

		return data, nil
	}

	if !provider.IsNotExist(err) {
		return nil, fmt.Errorf("unable to read: %s", err)
	}

	exif, err := a.fetchAndStoreExif(item)
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

	a.exifCounter.WithLabelValues("requested").Inc()

	resp, err := request.New().WithClient(exasClient).Post(a.exifURL).Send(context.Background(), file)
	if err != nil {
		return nil, err
	}

	var exifs []map[string]interface{}
	if err := httpjson.Read(resp, &exifs); err != nil {
		return nil, fmt.Errorf("unable to read exas response: %s", err)
	}

	writer, err := a.storageApp.WriterTo(getExifPath(item, ""))
	if err != nil {
		return nil, fmt.Errorf("unable to get writer: %s", err)
	}

	defer func() {
		if err := writer.Close(); err != nil {
			logger.Error("unable to close exif file: %s", err)
		}
	}()

	var data map[string]interface{}
	if len(exifs) > 0 {
		data = exifs[0]
	}

	if err := json.NewEncoder(writer).Encode(data); err != nil {
		return nil, fmt.Errorf("unable to encode: %s", err)
	}

	a.exifCounter.WithLabelValues("saved").Inc()

	return data, nil
}

func (a app) UpdateDate(item provider.StorageItem) {
	data, err := a.get(item)
	if err != nil {
		logger.Error("unable to retrieve exif for `%s`: %s", item.Pathname, err)
		return
	}

	if data == nil {
		return
	}

	for _, exifDate := range exifDates {
		rawCreateDate, ok := data[exifDate]
		if !ok {
			continue
		}

		createDateStr, ok := rawCreateDate.(string)
		if !ok {
			logger.Error("key `%s` is not a string for `%s`", exifDate, item.Pathname)
			return
		}

		createDate, err := parseDate(createDateStr)
		if err != nil {
			logger.Error("unable to parse `%s` with value `%s` for `%s`: %s", exifDate, createDateStr, item.Pathname, err)
			return
		}

		if createDate.IsZero() {
			a.dateCounter.WithLabelValues("zero").Inc()
			return
		}

		if item.Date.Equal(createDate) {
			a.dateCounter.WithLabelValues("equal").Inc()
			return
		}

		a.dateCounter.WithLabelValues("updated").Inc()

		if err := a.storageApp.UpdateDate(item.Pathname, createDate); err != nil {
			logger.Error("unable to update date for `%s`: %s", item.Pathname, err)
		}

		return
	}
}

func parseDate(raw string) (time.Time, error) {
	for _, pattern := range datePatterns {
		createDate, err := time.Parse(pattern, raw)
		if err == nil {
			return createDate, nil
		}
	}

	return time.Time{}, errors.New("no matching pattern")
}

func (a app) Rename(old, new provider.StorageItem) {
	if !a.enabled() {
		return
	}

	for _, suffix := range exifSuffixes {
		oldPath := getExifPath(old, suffix)
		if _, err := a.storageApp.Info(oldPath); provider.IsNotExist(err) {
			return
		}

		if err := a.storageApp.Rename(oldPath, getExifPath(new, suffix)); err != nil {
			logger.Error("unable to rename exif: %s", err)
		}
	}
}

func (a app) Delete(item provider.StorageItem) {
	if !a.enabled() {
		return
	}

	for _, suffix := range exifSuffixes {
		if err := a.storageApp.Remove(getExifPath(item, suffix)); err != nil {
			logger.Error("unable to delete exif: %s", err)
		}
	}
}
