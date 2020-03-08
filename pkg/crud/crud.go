package crud

import (
	"errors"
	"flag"
	"mime/multipart"
	"net/http"
	"regexp"
	"strings"
	"sync"

	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/fibr/pkg/thumbnail"
	"github.com/ViBiOh/httputils/v3/pkg/flags"
	"github.com/ViBiOh/httputils/v3/pkg/logger"
)

var (
	// ErrNotAuthorized error returned when user is not authorized
	ErrNotAuthorized = errors.New("you're not authorized to do this ⛔")

	// ErrEmptyName error returned when user does not provide a name
	ErrEmptyName = errors.New("provided name is empty")
)

// App of package
type App interface {
	Start()

	Browser(http.ResponseWriter, provider.Request, provider.StorageItem, *provider.Message)
	ServeStatic(http.ResponseWriter, *http.Request) bool

	List(http.ResponseWriter, provider.Request, *provider.Message)
	Get(http.ResponseWriter, *http.Request, provider.Request)
	Post(http.ResponseWriter, *http.Request, provider.Request)
	Create(http.ResponseWriter, *http.Request, provider.Request)
	Upload(http.ResponseWriter, *http.Request, provider.Request, *multipart.Part)
	Rename(http.ResponseWriter, *http.Request, provider.Request)
	Delete(http.ResponseWriter, *http.Request, provider.Request)

	GetShare(string) *provider.Share
	CreateShare(http.ResponseWriter, *http.Request, provider.Request)
	DeleteShare(http.ResponseWriter, *http.Request, provider.Request)
}

// Config of package
type Config struct {
	metadata        *bool
	ignore          *string
	sanitizeOnStart *bool
}

type app struct {
	metadataEnabled bool
	metadatas       []*provider.Share
	metadataLock    sync.Mutex
	sanitizeOnStart bool

	storage   provider.Storage
	renderer  provider.Renderer
	thumbnail thumbnail.App
}

// Flags adds flags for configuring package
func Flags(fs *flag.FlagSet, prefix string) Config {
	return Config{
		metadata:        flags.New(prefix, "crud").Name("Metadata").Default(true).Label("Enable metadata storage").ToBool(fs),
		ignore:          flags.New(prefix, "crud").Name("IgnorePattern").Default("").Label("Ignore pattern when listing files or directory").ToString(fs),
		sanitizeOnStart: flags.New(prefix, "crud").Name("SanitizeOnStart").Default(false).Label("Sanitize name on start").ToBool(fs),
	}
}

// New creates new App from Config
func New(config Config, storage provider.Storage, renderer provider.Renderer, thumbnail thumbnail.App) (App, error) {
	app := &app{
		metadataEnabled: *config.metadata,
		metadataLock:    sync.Mutex{},
		sanitizeOnStart: *config.sanitizeOnStart,

		storage:   storage,
		renderer:  renderer,
		thumbnail: thumbnail,
	}

	if app.metadataEnabled {
		logger.Fatal(app.loadMetadata())
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

func (a *app) Start() {
	err := a.storage.Walk("", func(item provider.StorageItem, _ error) error {
		if name, err := provider.SanitizeName(item.Pathname, false); err != nil {
			logger.Error("unable to sanitize name %s: %s", item.Pathname, err)
		} else if name != item.Pathname {
			if a.sanitizeOnStart {
				logger.Info("Renaming `%s` to `%s`", item.Pathname, name)
				if _, err := a.doRename(item.Pathname, name, item); err != nil {
					logger.Error("%s", err)
				}
			} else {
				logger.Info("File with name `%s` should be renamed to `%s`", item.Pathname, name)
			}
		}

		if thumbnail.CanHaveThumbnail(item) && !a.thumbnail.HasThumbnail(item) {
			a.thumbnail.GenerateThumbnail(item)
		}

		return nil
	})

	if err != nil {
		logger.Error("%s", err)
	}
}

// GetShare returns share configuration if request path match
func (a *app) GetShare(requestPath string) *provider.Share {
	cleanPath := strings.TrimPrefix(requestPath, "/")

	for _, share := range a.metadatas {
		if strings.HasPrefix(cleanPath, share.ID) {
			return share
		}
	}

	return nil
}
