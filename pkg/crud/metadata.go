package crud

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"path"

	"github.com/ViBiOh/fibr/pkg/provider"
)

var (
	metadataFilename = path.Join(provider.MetadataDirectoryName, ".json")
)

func (a *app) loadMetadata() error {
	_, err := a.storage.Info(metadataFilename)
	if err != nil && !provider.IsNotExist(err) {
		return err
	}

	if provider.IsNotExist(err) {
		if err := a.storage.CreateDir(provider.MetadataDirectoryName); err != nil {
			return err
		}

		a.metadatas = make([]*provider.Share, 0)

		return nil
	}

	file, err := a.storage.ReaderFrom(metadataFilename)
	if file != nil {
		defer func() {
			if closeErr := file.Close(); closeErr != nil {
				err = fmt.Errorf("%s: %w", err, closeErr)
			}
		}()
	}
	if err != nil {
		return err
	}

	rawMeta, err := ioutil.ReadAll(file)
	if err != nil {
		return err

	}

	if err = json.Unmarshal(rawMeta, &a.metadatas); err != nil {
		return err
	}

	return nil
}

func (a *app) saveMetadata() (err error) {
	if !a.metadataEnabled {
		return errors.New("metadata not enabled")
	}

	content, err := json.MarshalIndent(&a.metadatas, "", "  ")
	if err != nil {
		return err
	}

	file, err := a.storage.WriterTo(metadataFilename)
	if file != nil {
		defer func() {
			if closeErr := file.Close(); closeErr != nil {
				err = fmt.Errorf("%s: %w", err, closeErr)
			}
		}()
	}
	if err != nil {
		return err
	}

	n, err := file.Write(content)
	if err != nil {
		return err
	}

	if n < len(content) {
		return io.ErrShortWrite
	}

	return nil
}
