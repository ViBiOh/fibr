package crud

import (
	"errors"
	"flag"
	"mime/multipart"
	"net/http"
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
	metadata *bool
}

type app struct {
	metadataEnabled bool
	metadatas       []*provider.Share
	metadataLock    sync.Mutex

	storage   provider.Storage
	renderer  provider.Renderer
	thumbnail thumbnail.App
}

// Flags adds flags for configuring package
func Flags(fs *flag.FlagSet, prefix string) Config {
	return Config{
		metadata: flags.New(prefix, "crud").Name("Metadata").Default(true).Label("Enable metadata storage").ToBool(fs),
	}
}

// New creates new App from Config
func New(config Config, storage provider.Storage, renderer provider.Renderer, thumbnail thumbnail.App) App {
	app := &app{
		metadataEnabled: *config.metadata,
		metadataLock:    sync.Mutex{},
		storage:         storage,
		renderer:        renderer,
		thumbnail:       thumbnail,
	}

	if app.metadataEnabled {
		logger.Fatal(app.loadMetadata())
	}

	return app
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
