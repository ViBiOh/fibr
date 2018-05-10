package crud

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path"

	"github.com/ViBiOh/fibr/pkg/provider"
)

var metadataFilename = []byte(`.json`)

func (a *App) loadMetadata() error {
	filename, info := a.getMetadataFileinfo(nil, metadataFilename)
	if info == nil {
		if err := os.MkdirAll(path.Dir(filename), 0700); err != nil {
			return fmt.Errorf(`Error while creating metadata dir: %v`, err)
		}

		a.metadatas = make([]*provider.Share, 0)

		return nil
	}

	rawMeta, err := ioutil.ReadFile(filename)
	if err != nil {
		return fmt.Errorf(`Error while reading metadata: %v`, err)

	}

	if err = json.Unmarshal(rawMeta, &a.metadatas); err != nil {
		return fmt.Errorf(`Error while unmarshalling metadata: %v`, err)
	}

	return nil
}

func (a *App) saveMetadata() error {
	if !a.metadataEnabled {
		return fmt.Errorf(`Metadatas not enabled`)
	}

	content, err := json.MarshalIndent(&a.metadatas, ``, `  `)
	if err != nil {
		return fmt.Errorf(`Error while marshalling metadata: %v`, err)
	}

	filename, _ := a.getMetadataFileinfo(nil, metadataFilename)
	if err := ioutil.WriteFile(filename, content, 0600); err != nil {
		return fmt.Errorf(`Error while writing metadatas: %v`, err)
	}

	return nil
}
