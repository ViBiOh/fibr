package crud

import (
	"embed"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"mime/multipart"
	"net/http"
	"regexp"
	"strings"

	"github.com/ViBiOh/fibr/pkg/metadata"
	"github.com/ViBiOh/fibr/pkg/provider"
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

//go:embed static
var filesystem embed.FS

// App of package
type App interface {
	Start(done <-chan struct{})

	Browser(http.ResponseWriter, provider.Request, provider.StorageItem, renderer.Message)
	ServeStatic(http.ResponseWriter, *http.Request) bool

	List(http.ResponseWriter, provider.Request, renderer.Message)
	Get(http.ResponseWriter, *http.Request, provider.Request)
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
	rendererApp  provider.Renderer
	metadataApp  metadata.App
	thumbnailApp thumbnail.App

	staticHandler   http.Handler
	publicURL       string
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
func New(config Config, storage provider.Storage, renderer provider.Renderer, metadata metadata.App, thumbnail thumbnail.App, prometheus prometheus.Registerer, publicURL string) (App, error) {
	app := &app{
		sanitizeOnStart: *config.sanitizeOnStart,
		publicURL:       publicURL,

		storageApp:   storage,
		rendererApp:  renderer,
		thumbnailApp: thumbnail,
		metadataApp:  metadata,
		prometheus:   prometheus,
	}

	staticFS, err := fs.Sub(filesystem, "static")
	if err != nil {
		return nil, fmt.Errorf("unable to get static/ filesystem: %s", err)
	}
	app.staticHandler = http.FileServer(http.FS(staticFS))

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

	err := a.storageApp.Walk("", func(item provider.StorageItem, _ error) error {
		name, err := provider.SanitizeName(item.Pathname, false)
		if err != nil {
			logger.Error("unable to sanitize name %s: %s", item.Pathname, err)
			return nil
		}

		if name == item.Pathname {
			return nil
		}

		if a.sanitizeOnStart {
			a.rename(item, name, renameCount)
		} else {
			logger.Info("File with name `%s` should be renamed to `%s`", item.Pathname, name)
		}

		if thumbnail.CanHaveThumbnail(item) && !a.thumbnailApp.HasThumbnail(item) {
			a.thumbnailApp.GenerateThumbnail(item)
		}

		return nil
	})

	if err != nil {
		logger.Error("%s", err)
	}
}

func (a *app) rename(item provider.StorageItem, name string, guage *prometheus.GaugeVec) {
	logger.Info("Renaming `%s` to `%s`", item.Pathname, name)

	if renamedItem, err := a.doRename(item.Pathname, name, item); err != nil {
		guage.WithLabelValues("error").Add(1.0)
		logger.Error("%s", err)
	} else {
		guage.WithLabelValues("success").Add(1.0)
		item = renamedItem
	}
}
