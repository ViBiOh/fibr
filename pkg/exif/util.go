package exif

import (
	"fmt"
	"path"
	"path/filepath"

	"github.com/ViBiOh/exas/pkg/model"
	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/httputils/v4/pkg/sha"
)

// CanHaveExif determine if exif can be extracted for given pathname
func (a App) CanHaveExif(item provider.StorageItem) bool {
	return (item.IsImage() || item.IsVideo() || item.IsPdf()) && (a.maxSize == 0 || item.Size < a.maxSize || a.directAccess)
}

func getExifPath(item provider.StorageItem) string {
	fullPath := path.Join(provider.MetadataDirectoryName, item.Pathname)
	if item.IsDir {
		return fmt.Sprintf("%s/%s.json", fullPath, "aggregate")
	}

	return fmt.Sprintf("%s/%s.json", filepath.Dir(fullPath), sha.New(item.Name))
}

func (a App) hasMetadata(item provider.StorageItem) bool {
	_, err := a.storageApp.Info(getExifPath(item))
	return err == nil
}

func (a App) loadExif(item provider.StorageItem) (model.Exif, error) {
	var data model.Exif
	return data, a.loadMetadata(item, &data)
}

func (a App) loadAggregate(item provider.StorageItem) (provider.Aggregate, error) {
	var data provider.Aggregate
	return data, a.loadMetadata(item, &data)
}

func (a App) loadMetadata(item provider.StorageItem, content interface{}) error {
	return provider.LoadJSON(a.storageApp, getExifPath(item), content)
}

func (a App) saveMetadata(item provider.StorageItem, data interface{}) error {
	filename := getExifPath(item)
	dirname := filepath.Dir(filename)

	if _, err := a.storageApp.Info(dirname); err != nil {
		if !provider.IsNotExist(err) {
			return fmt.Errorf("unable to check directory existence: %s", err)
		}

		if err = a.storageApp.CreateDir(dirname); err != nil {
			return fmt.Errorf("unable to create directory: %s", err)
		}
	}

	if err := provider.SaveJSON(a.storageApp, filename, data); err != nil {
		return fmt.Errorf("unable to save: %s", err)
	}

	if item.IsDir {
		a.increaseAggregate("save")
	} else {
		a.increaseExif("save")
	}

	return nil
}
