package thumbnail

import (
	"errors"
	"fmt"
	"path"
	"strings"

	"github.com/ViBiOh/fibr/pkg/provider"
)

// CanHaveThumbnail determine if thumbnail can be generated for given pathname
func (a App) CanHaveThumbnail(item provider.StorageItem) bool {
	return (item.IsImage() || item.IsPdf() || item.IsVideo()) && (a.maxSize == 0 || item.Size < a.maxSize)
}

// HasThumbnail determine if thumbnail exist for given pathname
func (a App) HasThumbnail(item provider.StorageItem) bool {
	if !a.enabled() {
		return false
	}

	if item.IsDir {
		return false
	}

	_, err := a.storageApp.Info(getThumbnailPath(item))
	return err == nil
}

// GetChunk retrieve the storage item in the metdata
func (a App) GetChunk(filename string) (provider.StorageItem, error) {
	if !a.enabled() {
		return provider.StorageItem{}, errors.New("thumbnail is disabled")
	}

	return a.storageApp.Info(path.Join(provider.MetadataDirectoryName, filename))
}

func getThumbnailPath(item provider.StorageItem) string {
	return getThumbnailExtension(item, "jpg")
}

func getStreamPath(item provider.StorageItem) string {
	return getThumbnailExtension(item, "m3u8")
}

func getThumbnailExtension(item provider.StorageItem, extension string) string {
	fullPath := path.Join(provider.MetadataDirectoryName, item.Pathname)
	if item.IsDir {
		return fullPath
	}

	return fmt.Sprintf("%s.%s", strings.TrimSuffix(fullPath, path.Ext(fullPath)), extension)
}
