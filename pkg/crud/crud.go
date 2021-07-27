package crud

import (
	"errors"
	"flag"
	"fmt"
	"mime/multipart"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/ViBiOh/fibr/pkg/exif"
	"github.com/ViBiOh/fibr/pkg/metadata"
	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/fibr/pkg/thumbnail"
	"github.com/ViBiOh/httputils/v4/pkg/flags"
	"github.com/ViBiOh/httputils/v4/pkg/logger"
	"github.com/ViBiOh/httputils/v4/pkg/renderer"
	"github.com/prometheus/client_golang/prometheus"
)

const (
	exifDate = "CreateDate"
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
	exifDateOnStart *bool
}

type app struct {
	prometheus   prometheus.Registerer
	storageApp   provider.Storage
	rendererApp  renderer.App
	metadataApp  metadata.App
	thumbnailApp thumbnail.App
	exifApp      exif.App

	sanitizeOnStart bool
	exifDateOnStart bool
}

// Flags adds flags for configuring package
func Flags(fs *flag.FlagSet, prefix string) Config {
	return Config{
		ignore:          flags.New(prefix, "crud").Name("IgnorePattern").Default("").Label("Ignore pattern when listing files or directory").ToString(fs),
		sanitizeOnStart: flags.New(prefix, "crud").Name("SanitizeOnStart").Default(false).Label("Sanitize name on start").ToBool(fs),
		exifDateOnStart: flags.New(prefix, "crud").Name("ExifDateOnStart").Default(false).Label("Change file date from EXIF date on start").ToBool(fs),
	}
}

// New creates new App from Config
func New(config Config, storage provider.Storage, rendererApp renderer.App, metadataApp metadata.App, thumbnailApp thumbnail.App, exifApp exif.App, prometheus prometheus.Registerer) (App, error) {
	app := &app{
		sanitizeOnStart: *config.sanitizeOnStart,
		exifDateOnStart: *config.exifDateOnStart,

		storageApp:   storage,
		rendererApp:  rendererApp,
		thumbnailApp: thumbnailApp,
		exifApp:      exifApp,
		metadataApp:  metadataApp,
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
	renameCount := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "fibr",
		Subsystem: "renames",
		Name:      "total",
	}, []string{"status"})
	if a.prometheus != nil {
		a.prometheus.MustRegister(renameCount)
	}

	err := a.storageApp.Walk("", func(item provider.StorageItem, err error) error {
		if err != nil {
			return err
		}

		item = a.sanitizeName(item, renameCount)

		if thumbnail.CanHaveThumbnail(item) && !a.thumbnailApp.HasThumbnail(item) {
			a.thumbnailApp.GenerateThumbnail(item)
		}

		if a.exifDateOnStart && exif.CanHaveExif(item) {
			if err := a.dateFromExif(item); err != nil {
				logger.Warn("unable to update date from exif for `%s`: %s", item.Pathname, err)
			}
		}

		return nil
	})

	if err != nil {
		logger.Error("%s", err)
	}
}

func (a *app) sanitizeName(item provider.StorageItem, renameCount *prometheus.GaugeVec) provider.StorageItem {
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

	return a.rename(item, name, renameCount)
}

func (a *app) rename(item provider.StorageItem, name string, guage *prometheus.GaugeVec) provider.StorageItem {
	logger.Info("Renaming `%s` to `%s`", item.Pathname, name)

	renamedItem, err := a.doRename(item.Pathname, name, item)
	if err != nil {
		guage.WithLabelValues("error").Add(1.0)
		logger.Error("%s", err)
		return item
	}

	guage.WithLabelValues("success").Add(1.0)
	return renamedItem
}

func (a *app) dateFromExif(item provider.StorageItem) error {
	data, err := a.exifApp.Get(item)
	if err != nil {
		return fmt.Errorf("unable to get exif data: %s", err)
	}

	rawCreateDate, ok := data[exifDate]
	if !ok {
		return fmt.Errorf("no `%s` found", exifDate)
	}

	createDateStr, ok := rawCreateDate.(string)
	if !ok {
		return fmt.Errorf("key `%s` is not a string", exifDate)
	}

	createDate, err := time.Parse("2006:01:02 15:04:05", createDateStr)
	if err != nil {
		return fmt.Errorf("unable to parse `%s`: %s", exifDate, err)
	}

	return a.storageApp.UpdateDate(item.Pathname, createDate)
}
