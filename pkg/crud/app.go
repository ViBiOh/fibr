package crud

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"path"
	"strings"
	"sync"

	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/fibr/pkg/thumbnail"
	"github.com/ViBiOh/fibr/pkg/utils"
	"github.com/ViBiOh/httputils/pkg/tools"
)

var (
	// ErrNotAuthorized error returned when user is not authorized
	ErrNotAuthorized = errors.New(`You're not authorized to do this ⛔`)

	// ErrEmptyName error returned when user does not provide a name
	ErrEmptyName = errors.New(`Provided name is empty`)
)

// App stores informations and secret of API
type App struct {
	rootDirectory   string
	rootDirname     string
	metadataEnabled bool
	metadatas       []*provider.Share
	metadataLock    sync.Mutex
	storage         provider.Storage
	renderer        provider.Renderer
	thumbnailApp    *thumbnail.App
}

// NewApp creates new App from Flags' config
func NewApp(config map[string]interface{}, storage provider.Storage, renderer provider.Renderer, thumbnailApp *thumbnail.App) *App {
	rootDirectory := *config[`directory`].(*string)
	_, info := utils.GetPathInfo(rootDirectory)
	if info == nil || !info.IsDir() {
		log.Fatalf(`Directory %s is unreachable`, rootDirectory)
	}

	app := &App{
		rootDirectory:   rootDirectory,
		rootDirname:     path.Base(rootDirectory),
		metadataEnabled: *config[`metadata`].(*bool),
		metadataLock:    sync.Mutex{},
		storage:         storage,
		renderer:        renderer,
		thumbnailApp:    thumbnailApp,
	}

	log.Printf(`Serving file from %s`, rootDirectory)

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
		`directory`: flag.String(tools.ToCamel(fmt.Sprintf(`%sDirectory`, prefix)), `/data`, `Directory to serve`),
		`metadata`:  flag.Bool(tools.ToCamel(fmt.Sprintf(`%sMetadata`, prefix)), true, `Enable metadata storage`),
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
