package thumbnail

import (
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

func getThumbnailPath(item provider.StorageItem) string {
	fullPath := path.Join(provider.MetadataDirectoryName, item.Pathname)
	if item.IsDir {
		return fullPath
	}

	return fmt.Sprintf("%s.jpg", strings.TrimSuffix(fullPath, path.Ext(fullPath)))
}
