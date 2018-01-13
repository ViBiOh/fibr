package crud

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"github.com/ViBiOh/fibr/utils"
	"github.com/ViBiOh/httputils/tools"
)

type share struct {
	id     string
	path   string
	public bool
	edit   bool
}

type metadata struct {
	shared map[string]share
}

// App stores informations and secret of API
type App struct {
	rootInfo         os.FileInfo
	rootFilename     string
	metadataFilename string
	metadata         *metadata
}

// NewApp creates new App from Flags' config
func NewApp(config map[string]*string) *App {
	app := &App{
		rootFilename: *config[`directory`],
	}

	_, info := utils.GetPathInfo(app.rootFilename)
	if info == nil || !info.IsDir() {
		log.Fatalf(`Directory %s is unreachable`, app.rootFilename)
	}

	app.rootInfo = info

	if err := app.loadMetadata(); err != nil {
		log.Printf(`Error while loading metadata: %v`, err)
	}

	return app
}

// Flags add flags for given prefix
func Flags(prefix string) map[string]*string {
	return map[string]*string{
		`directory`: flag.String(tools.ToCamel(prefix+`Directory`), `/data`, `Directory to serve`),
	}
}

// GetRootAbsName returns absolute name of root directory served
func (a *App) GetRootAbsName() string {
	return a.rootFilename
}

// GetRootName returns name of root directory served
func (a *App) GetRootName() string {
	return a.rootInfo.Name()
}

func (a *App) loadMetadata() error {
	filename, info := utils.GetPathInfo(a.rootFilename, `.fibr.json`)
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
