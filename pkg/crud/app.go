package crud

import (
	"errors"
	"flag"
	"fmt"
	"strings"
	"sync"

	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/fibr/pkg/thumbnail"
	"github.com/ViBiOh/httputils/pkg/logger"
	"github.com/ViBiOh/httputils/pkg/tools"
)

var (
	// ErrNotAuthorized error returned when user is not authorized
	ErrNotAuthorized = errors.New(`you're not authorized to do this ⛔`)

	// ErrEmptyName error returned when user does not provide a name
	ErrEmptyName = errors.New(`provided name is empty`)
)

// Config of package
type Config struct {
	metadata *bool
}

// App of package
type App struct {
	metadataEnabled bool
	metadatas       []*provider.Share
	metadataLock    sync.Mutex
	storage         provider.Storage
	renderer        provider.Renderer
	thumbnailApp    *thumbnail.App
}

// Flags adds flags for configuring package
func Flags(fs *flag.FlagSet, prefix string) Config {
	return Config{
		metadata: fs.Bool(tools.ToCamel(fmt.Sprintf(`%sMetadata`, prefix)), true, `Enable metadata storage`),
	}
}

// New creates new App from Config
func New(config Config, storage provider.Storage, renderer provider.Renderer, thumbnailApp *thumbnail.App) *App {
	app := &App{
		metadataEnabled: *config.metadata,
		metadataLock:    sync.Mutex{},
		storage:         storage,
		renderer:        renderer,
		thumbnailApp:    thumbnailApp,
	}

	if app.metadataEnabled {
		if err := app.loadMetadata(); err != nil {
			logger.Fatal(`%+v`, err)
		}

		go thumbnailApp.Generate()
	}

	return app
}

// GetShare returns share configuration if request path match
func (a *App) GetShare(requestPath string) *provider.Share {
	cleanPath := strings.TrimPrefix(requestPath, `/`)

	for _, share := range a.metadatas {
		if strings.HasPrefix(cleanPath, share.ID) {
			return share
		}
	}

	return nil
}
