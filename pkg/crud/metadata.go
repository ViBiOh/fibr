package crud

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"path"

	"github.com/ViBiOh/fibr/pkg/provider"
)

var (
	metadataFilename = path.Join(provider.MetadataDirectoryName, `.json`)
)

func (a *App) loadMetadata() (err error) {
	info, err := a.storage.Info(metadataFilename)
	if err != nil && !provider.IsNotExist(err) {
		return fmt.Errorf(`Error while getting metadata: %v`, err)
	}

	if info == nil {
		if err := a.storage.Create(provider.MetadataDirectoryName); err != nil {
			return fmt.Errorf(`Error while creating metadata: %v`, err)
		}

		a.metadatas = make([]*provider.Share, 0)

		return nil
	}

	file, err := a.storage.Read(metadataFilename)
	if file != nil {
		defer func() {
			if closeErr := file.Close(); closeErr != nil {
				err = fmt.Errorf(`%s, and also error while closing metadata: %v`, err, closeErr)
			}
		}()
	}
	if err != nil {
		return fmt.Errorf(`Error while opening metadata: %v`, err)
	}

	rawMeta, err := ioutil.ReadAll(file)
	if err != nil {
		return fmt.Errorf(`Error while reading metadata: %v`, err)

	}

	if err = json.Unmarshal(rawMeta, &a.metadatas); err != nil {
		return fmt.Errorf(`Error while unmarshalling metadata: %v`, err)
	}

	return nil
}

func (a *App) saveMetadata() (err error) {
	if !a.metadataEnabled {
		return fmt.Errorf(`Metadata not enabled`)
	}

	content, err := json.MarshalIndent(&a.metadatas, ``, `  `)
	if err != nil {
		return fmt.Errorf(`Error while marshalling metadata: %v`, err)
	}

	file, err := a.storage.Open(metadataFilename)
	if file != nil {
		defer func() {
			if closeErr := file.Close(); closeErr != nil {
				err = fmt.Errorf(`%s, and also error while closing metadata: %v`, err, closeErr)
			}
		}()
	}
	if err != nil {
		return fmt.Errorf(`Error while opening metadata: %v`, err)
	}

	n, err := file.Write(content)
	if err != nil {
		return fmt.Errorf(`Error while writing metadatas: %v`, err)
	}
	if n < len(content) {
		return fmt.Errorf(`Error while writing metadatas: %v`, io.ErrShortWrite)
	}

	return nil
}
