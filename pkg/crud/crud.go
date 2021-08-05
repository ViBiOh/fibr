package crud

import (
	"errors"
	"flag"
	"mime/multipart"
	"net/http"
	"regexp"
	"strings"

	"github.com/ViBiOh/fibr/pkg/exif"
	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/fibr/pkg/share"
	"github.com/ViBiOh/fibr/pkg/thumbnail"
	"github.com/ViBiOh/httputils/v4/pkg/flags"
	"github.com/ViBiOh/httputils/v4/pkg/logger"
	"github.com/ViBiOh/httputils/v4/pkg/renderer"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	// ErrNotAuthorized error returned when user is not authorized
	ErrNotAuthorized = errors.New("you're not authorized to do this ⛔")

	// ErrEmptyName error returned when user does not provide a name
	ErrEmptyName = errors.New("name is empty")

	// ErrEmptyFolder error returned when user does not provide a folder
	ErrEmptyFolder = errors.New("folder is empty")

	// ErrAbsoluteFolder error returned when user provide a relative folder
	ErrAbsoluteFolder = errors.New("folder has to be absolute")
)

// App of package
type App interface {
	Start(done <-chan struct{})

	Browser(http.ResponseWriter, provider.Request, provider.StorageItem, renderer.Message) (string, int, map[string]interface{}, error)
	List(http.ResponseWriter, provider.Request, renderer.Message) (string, int, map[string]interface{}, error)
	Get(http.ResponseWriter, *http.Request, provider.Request) (string, int, map[string]interface{}, error)

	Post(http.ResponseWriter, *http.Request, provider.Request)
	Create(http.ResponseWriter, *http.Request, provider.Request)
	Upload(http.ResponseWriter, *http.Request, provider.Request, map[string]string, *multipart.Part)
	Rename(http.ResponseWriter, *http.Request, provider.Request)
	Delete(http.ResponseWriter, *http.Request, provider.Request)

	CreateShare(http.ResponseWriter, *http.Request, provider.Request)
	DeleteShare(http.ResponseWriter, *http.Request, provider.Request)
}

// Config of package
type Config struct {
	ignore          *string
	sanitizeOnStart *bool
}

type app struct {
	prometheus   prometheus.Registerer
	storageApp   provider.Storage
	rendererApp  renderer.App
	shareApp     share.App
	thumbnailApp thumbnail.App
	exifApp      exif.App

	pushEvent provider.EventProducer

	sanitizeOnStart bool
}

// Flags adds flags for configuring package
func Flags(fs *flag.FlagSet, prefix string) Config {
	return Config{
		ignore:          flags.New(prefix, "crud").Name("IgnorePattern").Default("").Label("Ignore pattern when listing files or directory").ToString(fs),
		sanitizeOnStart: flags.New(prefix, "crud").Name("SanitizeOnStart").Default(false).Label("Sanitize name on start").ToBool(fs),
	}
}

// New creates new App from Config
func New(config Config, storage provider.Storage, rendererApp renderer.App, shareApp share.App, thumbnailApp thumbnail.App, exifApp exif.App, prometheus prometheus.Registerer, eventProducer provider.EventProducer) (App, error) {
	app := &app{
		sanitizeOnStart: *config.sanitizeOnStart,

		pushEvent: eventProducer,

		storageApp:   storage,
		rendererApp:  rendererApp,
		thumbnailApp: thumbnailApp,
		exifApp:      exifApp,
		shareApp:     shareApp,
		prometheus:   prometheus,
	}

	var ignorePattern *regexp.Regexp
	ignore := strings.TrimSpace(*config.ignore)
	if len(ignore) != 0 {
		pattern, err := regexp.Compile(ignore)
		if err != nil {
			return nil, err
		}

		ignorePattern = pattern
		logger.Info("Ignoring files with pattern `%s`", ignore)
	}

	storage.SetIgnoreFn(func(item provider.StorageItem) bool {
		if item.IsDir && item.Name == provider.MetadataDirectoryName {
			return true
		}

		if ignorePattern != nil && ignorePattern.MatchString(item.Name) {
			return true
		}

		return false
	})

	return app, nil
}

func (a *app) Start(done <-chan struct{}) {
	sanitizeCounter := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "fibr",
		Subsystem: "sanitize",
		Name:      "total",
	}, []string{"status"})

	if a.prometheus != nil {
		if err := a.prometheus.Register(sanitizeCounter); err != nil {
			logger.Error("unable to register sanitize count metric: %s", err)
		}
	}

	err := a.storageApp.Walk("", func(item provider.StorageItem, err error) error {
		if err != nil {
			return err
		}

		select {
		case <-done:
			return errors.New("server is shutting down")
		default:
		}

		item = a.sanitizeName(item, sanitizeCounter)
		a.notify(provider.NewStartEvent(item))

		return nil
	})

	if err != nil {
		logger.Error("%s", err)
	}
}

func (a *app) sanitizeName(item provider.StorageItem, gauge *prometheus.GaugeVec) provider.StorageItem {
	name, err := provider.SanitizeName(item.Pathname, false)
	if err != nil {
		logger.Error("unable to sanitize name %s: %s", item.Pathname, err)
		return item
	}

	if name == item.Pathname {
		return item
	}

	if !a.sanitizeOnStart {
		logger.Info("File with name `%s` should be renamed to `%s`", item.Pathname, name)
		return item
	}

	return a.rename(item, name, gauge)
}

func (a *app) rename(item provider.StorageItem, name string, gauge *prometheus.GaugeVec) provider.StorageItem {
	logger.Info("Renaming `%s` to `%s`", item.Pathname, name)

	renamedItem, err := a.doRename(item.Pathname, name, item)
	if err != nil {
		gauge.WithLabelValues("error").Inc()
		logger.Error("%s", err)
		return item
	}

	gauge.WithLabelValues("success").Inc()
	return renamedItem
}
