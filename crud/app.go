package crud

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"strings"
	"sync"

	"github.com/ViBiOh/fibr/provider"
	"github.com/ViBiOh/fibr/utils"
	"github.com/ViBiOh/httputils/tools"
)

// Share stores informations about shared paths
type Share struct {
	ID   string `json:"id"`
	Path string `json:"path"`
	Edit bool   `json:"edit"`
}

// App stores informations and secret of API
type App struct {
	rootDirectory    string
	metadataFilename string
	metadatas        []*Share
	metadataLock     sync.Mutex
	renderer         provider.Renderer
}

// NewApp creates new App from Flags' config
func NewApp(config map[string]*string, renderer provider.Renderer) *App {
	app := &App{
		rootDirectory: *config[`directory`],
		renderer:      renderer,
		metadataLock:  sync.Mutex{},
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
func Flags(prefix string) map[string]*string {
	return map[string]*string{
		`directory`: flag.String(tools.ToCamel(fmt.Sprintf(`%sDirectory`, prefix)), `/data`, `Directory to serve`),
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
		return nil
	}

	rawMeta, err := ioutil.ReadFile(filename)
	if err != nil {
		return fmt.Errorf(`Error while reading metadata: %v`, err)
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

	content, err := json.Marshal(&a.metadatas)
	if err != nil {
		return fmt.Errorf(`Error while marshalling metadata: %v`, err)
	}

	if err := ioutil.WriteFile(a.metadataFilename, content, 0600); err != nil {
		return fmt.Errorf(`Error while writing metadatas: %v`, err)
	}

	return nil
}
