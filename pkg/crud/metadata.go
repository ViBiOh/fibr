package crud

import (
	"encoding/json"
	"io"
	"io/ioutil"
	"path"

	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/httputils/pkg/errors"
)

var (
	metadataFilename = path.Join(provider.MetadataDirectoryName, ".json")
)

func (a *App) loadMetadata() (err error) {
	info, err := a.storage.Info(metadataFilename)
	if err != nil && !provider.IsNotExist(err) {
		return err
	}

	if info == nil {
		if err := a.storage.Create(provider.MetadataDirectoryName); err != nil {
			return err
		}

		a.metadatas = make([]*provider.Share, 0)

		return nil
	}

	file, err := a.storage.Read(metadataFilename)
	if file != nil {
		defer func() {
			if closeErr := file.Close(); closeErr != nil {
				err = errors.New("%s and also %v", err, closeErr)
			}
		}()
	}
	if err != nil {
		return err
	}

	rawMeta, err := ioutil.ReadAll(file)
	if err != nil {
		return errors.WithStack(err)

	}

	if err = json.Unmarshal(rawMeta, &a.metadatas); err != nil {
		return errors.WithStack(err)
	}

	return nil
}

func (a *App) saveMetadata() (err error) {
	if !a.metadataEnabled {
		return errors.New("metadata not enabled")
	}

	content, err := json.MarshalIndent(&a.metadatas, "", "  ")
	if err != nil {
		return errors.WithStack(err)
	}

	file, err := a.storage.Open(metadataFilename)
	if file != nil {
		defer func() {
			if closeErr := file.Close(); closeErr != nil {
				err = errors.New("%s and also %v", err, closeErr)
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
		return errors.WithStack(io.ErrShortWrite)
	}

	return nil
}
