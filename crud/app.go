package crud

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"sync"

	"github.com/ViBiOh/fibr/provider"
	"github.com/ViBiOh/fibr/utils"
	"github.com/ViBiOh/httputils/tools"
)

// ErrNotAuthorized error returned when user is not authorized
var ErrNotAuthorized = errors.New(`You're not authorized to do this ⛔`)

// Share stores informations about shared paths
type Share struct {
	ID   string `json:"id"`
	Path string `json:"path"`
	Edit bool   `json:"edit"`
}

// App stores informations and secret of API
type App struct {
	rootDirectory    string
	metadataCreate   bool
	metadataFilename string
	metadatas        []*Share
	metadataLock     sync.Mutex
	renderer         provider.Renderer
}

// NewApp creates new App from Flags' config
func NewApp(config map[string]interface{}, renderer provider.Renderer) *App {
	app := &App{
		rootDirectory:  *config[`directory`].(*string),
		metadataCreate: *config[`createMeta`].(*bool),
		renderer:       renderer,
		metadataLock:   sync.Mutex{},
	}

	log.Printf(`Serving file from %s`, app.rootDirectory)

	_, info := utils.GetPathInfo(app.rootDirectory)
	if info == nil || !info.IsDir() {
		log.Fatalf(`Directory %s is unreachable`, app.rootDirectory)
	}

	if err := app.loadMetadata(); err != nil {
		log.Fatalf(`Error while loading metadata: %v`, err)
	}

	go app.generateThumbnail()

	return app
}

// Flags add flags for given prefix
func Flags(prefix string) map[string]interface{} {
	return map[string]interface{}{
		`directory`:  flag.String(tools.ToCamel(fmt.Sprintf(`%sDirectory`, prefix)), `/data`, `Directory to serve`),
		`createMeta`: flag.Bool(tools.ToCamel(fmt.Sprintf(`%sCreateMeta`, prefix)), false, `Create metadata directory if not exist`),
	}
}

// GetRootDirectory returns absolute name of root directory served
func (a *App) GetRootDirectory() string {
	return a.rootDirectory
}

// GetSharedPath returns share configurion if request path match
func (a *App) GetSharedPath(requestPath string) *Share {
	cleanPath := strings.TrimPrefix(requestPath, `/`)

	for _, share := range a.metadatas {
		if strings.HasPrefix(cleanPath, share.ID) {
			return share
		}
	}

	return nil
}

func (a *App) loadMetadata() error {
	filename, info := utils.GetPathInfo(a.rootDirectory, provider.MetadataDirectoryName, `.json`)
	if info == nil {
		if !a.metadataCreate {
			return nil
		}

		if err := os.MkdirAll(filename, 0700); err != nil {
			return fmt.Errorf(`Error while creating metadata dir: %v`, err)
		}
	}

	rawMeta, err := ioutil.ReadFile(filename)
	if err != nil {
		if !a.metadataCreate {
			return fmt.Errorf(`Error while reading metadata: %v`, err)
		}

		rawMeta = []byte(`{}`)
	}

	if err = json.Unmarshal(rawMeta, &a.metadatas); err != nil {
		return fmt.Errorf(`Error while unmarshalling metadata: %v`, err)
	}

	a.metadataFilename = filename

	return nil
}

func (a *App) saveMetadata() error {
	if a.metadataFilename == `` {
		return fmt.Errorf(`No metadata file loaded`)
	}

	content, err := json.MarshalIndent(&a.metadatas, ``, `  `))
	if err != nil {
		return fmt.Errorf(`Error while marshalling metadata: %v`, err)
	}

	if err := ioutil.WriteFile(a.metadataFilename, content, 0600); err != nil {
		return fmt.Errorf(`Error while writing metadatas: %v`, err)
	}

	return nil
}
