package exif

import (
	"encoding/json"
	"fmt"
	"path"
	"strings"

	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/httputils/v4/pkg/logger"
)

// CanHaveExif determine if exif can be extracted for given pathname
func CanHaveExif(item provider.StorageItem) bool {
	return (item.IsImage() || item.IsVideo() || item.IsPdf()) && item.Size < maxExifSize
}

func getExifPath(item provider.StorageItem, suffix string) string {
	fullPath := path.Join(provider.MetadataDirectoryName, item.Pathname)
	if item.IsDir {
		fullPath += "/"
	}

	name := strings.TrimSuffix(fullPath, path.Ext(fullPath))

	if len(suffix) == 0 {
		return fmt.Sprintf("%s.json", name)
	}

	return fmt.Sprintf("%s_%s.json", name, suffix)
}

func (a app) HasExif(item provider.StorageItem) bool {
	return a.hasMetadata(item, exifMetadataFilename)
}

func (a app) HasGeocode(item provider.StorageItem) bool {
	return a.hasMetadata(item, geocodeMetadataFilename)
}

func (a app) HasAggregat(item provider.StorageItem) bool {
	return a.hasMetadata(item, aggregateMetadataFilename)
}

func (a app) hasMetadata(item provider.StorageItem, suffix string) bool {
	if !a.enabled() {
		return false
	}

	_, err := a.storageApp.Info(getExifPath(item, suffix))
	return err == nil
}

func (a app) loadExif(item provider.StorageItem) (map[string]interface{}, error) {
	var data map[string]interface{}
	return data, a.loadMetadata(item, exifMetadataFilename, &data)
}

func (a app) loadGeocode(item provider.StorageItem) (geocode, error) {
	var data geocode
	return data, a.loadMetadata(item, geocodeMetadataFilename, &data)
}

func (a app) loadAggregate(item provider.StorageItem) (provider.Aggregate, error) {
	var data provider.Aggregate
	return data, a.loadMetadata(item, aggregateMetadataFilename, &data)
}

func (a app) loadMetadata(item provider.StorageItem, suffix string, content interface{}) error {
	reader, err := a.storageApp.ReaderFrom(getExifPath(item, suffix))
	if err != nil {
		if !provider.IsNotExist(err) {
			return fmt.Errorf("unable to read: %s", err)
		}
		return nil
	}

	if err := json.NewDecoder(reader).Decode(content); err != nil {
		return fmt.Errorf("unable to decode: %s", err)
	}

	return nil
}

func (a app) saveMetadata(item provider.StorageItem, suffix string, data interface{}) error {
	writer, err := a.storageApp.WriterTo(getExifPath(item, suffix))
	if err != nil {
		return fmt.Errorf("unable to get writer: %s", err)
	}

	defer func() {
		if err := writer.Close(); err != nil {
			logger.Error("unable to close file: %s", err)
		}
	}()

	if err := json.NewEncoder(writer).Encode(data); err != nil {
		return fmt.Errorf("unable to encode: %s", err)
	}

	return nil
}
