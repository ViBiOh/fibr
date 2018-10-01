package crud

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"strings"
	"sync"

	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/fibr/pkg/thumbnail"
	"github.com/ViBiOh/httputils/pkg/tools"
)

var (
	// ErrNotAuthorized error returned when user is not authorized
	ErrNotAuthorized = errors.New(`you're not authorized to do this ⛔`)

	// ErrEmptyName error returned when user does not provide a name
	ErrEmptyName = errors.New(`provided name is empty`)
)

// App stores informations and secret of API
type App struct {
	metadataEnabled bool
	metadatas       []*provider.Share
	metadataLock    sync.Mutex
	storage         provider.Storage
	renderer        provider.Renderer
	thumbnailApp    *thumbnail.App
}

// NewApp creates new App from Flags' config
func NewApp(config map[string]interface{}, storage provider.Storage, renderer provider.Renderer, thumbnailApp *thumbnail.App) *App {
	app := &App{
		metadataEnabled: *config[`metadata`].(*bool),
		metadataLock:    sync.Mutex{},
		storage:         storage,
		renderer:        renderer,
		thumbnailApp:    thumbnailApp,
	}

	if app.metadataEnabled {
		if err := app.loadMetadata(); err != nil {
			log.Fatalf(`Error while loading metadata: %v`, err)
		}

		go thumbnailApp.Generate()
	}

	return app
}

// Flags add flags for given prefix
func Flags(prefix string) map[string]interface{} {
	return map[string]interface{}{
		`metadata`: flag.Bool(tools.ToCamel(fmt.Sprintf(`%sMetadata`, prefix)), true, `Enable metadata storage`),
	}
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
