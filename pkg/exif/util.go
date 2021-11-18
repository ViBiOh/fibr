package exif

import (
	"fmt"
	"path"
	"path/filepath"
	"strings"

	"github.com/ViBiOh/exas/pkg/model"
	"github.com/ViBiOh/fibr/pkg/provider"
)

// CanHaveExif determine if exif can be extracted for given pathname
func (a App) CanHaveExif(item provider.StorageItem) bool {
	return (item.IsImage() || item.IsVideo() || item.IsPdf()) && (a.maxSize == 0 || item.Size < a.maxSize || a.directAccess)
}

func getExifPath(item provider.StorageItem, suffix string) string {
	fullPath := path.Join(provider.MetadataDirectoryName, item.Pathname)
	if item.IsDir {
		return fmt.Sprintf("%s/%s.json", fullPath, suffix)
	}

	name := strings.TrimSuffix(fullPath, path.Ext(fullPath))

	if len(suffix) == 0 {
		return fmt.Sprintf("%s.json", name)
	}

	return fmt.Sprintf("%s_%s.json", name, suffix)
}

func (a App) hasMetadata(item provider.StorageItem, suffix string) bool {
	if !a.enabled() {
		return false
	}

	_, err := a.storageApp.Info(getExifPath(item, suffix))
	return err == nil
}

func (a App) loadExif(item provider.StorageItem) (model.Exif, error) {
	var data model.Exif
	return data, a.loadMetadata(item, exifMetadataFilename, &data)
}

func (a App) loadAggregate(item provider.StorageItem) (provider.Aggregate, error) {
	var data provider.Aggregate
	return data, a.loadMetadata(item, aggregateMetadataFilename, &data)
}

func (a App) loadMetadata(item provider.StorageItem, suffix string, content interface{}) error {
	return provider.LoadJSON(a.storageApp, getExifPath(item, suffix), content)
}

func (a App) saveMetadata(item provider.StorageItem, suffix string, data interface{}) error {
	filename := getExifPath(item, suffix)
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

	switch suffix {
	case exifMetadataFilename:
		a.increaseExif("save")
	case aggregateMetadataFilename:
		a.increaseAggregate("save")
	}

	return nil
}
