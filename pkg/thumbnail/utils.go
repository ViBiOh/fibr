package thumbnail

import (
	"fmt"
	"path"
	"strings"

	"github.com/ViBiOh/fibr/pkg/provider"
)

const (
	maxThumbnailSize = 1024 * 1024 * 150 // 150mo
)

// CanHaveThumbnail determine if thumbnail can be generated for given pathname
func CanHaveThumbnail(item provider.StorageItem) bool {
	return (item.IsImage() || item.IsPdf() || item.IsVideo()) && item.Size < maxThumbnailSize
}

// HasThumbnail determine if thumbnail exist for given pathname
func (a app) HasThumbnail(item provider.StorageItem) bool {
	if !a.enabled() {
		return false
	}

	info, err := a.storageApp.Info(getThumbnailPath(item))
	return err == nil && !info.IsDir
}

func getThumbnailPath(item provider.StorageItem) string {
	fullPath := path.Join(provider.MetadataDirectoryName, item.Pathname)
	if item.IsDir {
		return fullPath
	}

	return fmt.Sprintf("%s.jpg", strings.TrimSuffix(fullPath, path.Ext(fullPath)))
}
