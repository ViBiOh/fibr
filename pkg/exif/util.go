package exif

import (
	"fmt"
	"path/filepath"

	absto "github.com/ViBiOh/absto/pkg/model"
	exas "github.com/ViBiOh/exas/pkg/model"
	"github.com/ViBiOh/fibr/pkg/provider"
)

// CanHaveExif determine if exif can be extracted for given pathname
func (a App) CanHaveExif(item absto.Item) bool {
	return provider.ThumbnailExtensions[item.Extension] && (a.maxSize == 0 || item.Size < a.maxSize || a.directAccess)
}

func getExifPath(item absto.Item) string {
	fullPath := provider.MetadataDirectoryName + item.Pathname
	if item.IsDir {
		return fullPath + "/aggregate.json"
	}

	return fmt.Sprintf("%s/%s.json", filepath.Dir(fullPath), item.ID)
}

func (a App) hasMetadata(item absto.Item) bool {
	_, err := a.storageApp.Info(getExifPath(item))
	return err == nil
}

func (a App) loadExif(item absto.Item) (exas.Exif, error) {
	var data exas.Exif
	return data, a.loadMetadata(item, &data)
}

func (a App) loadAggregate(item absto.Item) (provider.Aggregate, error) {
	var data provider.Aggregate
	return data, a.loadMetadata(item, &data)
}

func (a App) loadMetadata(item absto.Item, content interface{}) error {
	return provider.LoadJSON(a.storageApp, getExifPath(item), content)
}

func (a App) saveMetadata(item absto.Item, data interface{}) error {
	filename := getExifPath(item)
	dirname := filepath.Dir(filename)

	if _, err := a.storageApp.Info(dirname); err != nil {
		if !absto.IsNotExist(err) {
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
