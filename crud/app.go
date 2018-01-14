package crud

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"

	"github.com/ViBiOh/fibr/ui"
	"github.com/ViBiOh/fibr/utils"
	"github.com/ViBiOh/httputils/tools"
)

type share struct {
	Path   string `json:"path"`
	Public bool   `json:"public"`
	Edit   bool   `json:"edit"`
}

// App stores informations and secret of API
type App struct {
	rootDirectory    string
	metadataFilename string
	metadata         map[string]share
	uiApp            *ui.App
}

// NewApp creates new App from Flags' config
func NewApp(config map[string]*string, uiApp *ui.App) *App {
	app := &App{
		rootDirectory: *config[`directory`],
		uiApp:         uiApp,
	}

	log.Printf(`Serving file from %s`, app.rootDirectory)

	_, info := utils.GetPathInfo(app.rootDirectory)
	if info == nil || !info.IsDir() {
		log.Fatalf(`Directory %s is unreachable`, app.rootDirectory)
	}

	if err := app.loadMetadata(); err != nil {
		log.Fatalf(`Error while loading metadata: %v`, err)
	}

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

func (a *App) loadMetadata() error {
	filename, info := utils.GetPathInfo(a.rootDirectory, `.fibr.json`)
	if info == nil {
		return nil
	}

	rawMeta, err := ioutil.ReadFile(filename)
	if err != nil {
		return fmt.Errorf(`Error while reading metadata: %v`, err)
	}

	if err = json.Unmarshal(rawMeta, &a.metadata); err != nil {
		return fmt.Errorf(`Error while unmarshalling metadata: %v`, err)
	}

	a.metadataFilename = filename

	return nil
}

func (a *App) saveMetadata() error {
	if a.metadataFilename == `` {
		return fmt.Errorf(`No metadata file loaded`)
	}

	content, err := json.Marshal(&a.metadata)
	if err != nil {
		return fmt.Errorf(`Error while marshalling metadata: %v`, err)
	}

	if err := ioutil.WriteFile(a.metadataFilename, content, 0600); err != nil {
		return fmt.Errorf(`Error while writing metadata: %v`, err)
	}

	return nil
}
