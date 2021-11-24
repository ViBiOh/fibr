package thumbnail

import (
	"fmt"
	"path"
	"path/filepath"

	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/httputils/v4/pkg/sha"
	"github.com/ViBiOh/vith/pkg/model"
)

// CanHaveThumbnail determine if thumbnail can be generated for given pathname
func (a App) CanHaveThumbnail(item provider.StorageItem) bool {
	return !item.IsDir && (item.IsImage() || item.IsPdf() || item.IsVideo()) && (a.maxSize == 0 || item.Size < a.maxSize || a.directAccess)
}

// HasThumbnail determine if thumbnail exist for given pathname
func (a App) HasThumbnail(item provider.StorageItem) bool {
	if item.IsDir {
		return false
	}

	_, err := a.storageApp.Info(getThumbnailPath(item))
	return err == nil
}

// GetChunk retrieve the storage item in the metadata
func (a App) GetChunk(pathname string) (provider.StorageItem, error) {
	return a.storageApp.Info(path.Join(provider.MetadataDirectoryName, pathname))
}

func getThumbnailPath(item provider.StorageItem) string {
	return getPathWithExtension(item, "webp")
}

func getStreamPath(item provider.StorageItem) string {
	return getPathWithExtension(item, "m3u8")
}

func getPathWithExtension(item provider.StorageItem, extension string) string {
	return fmt.Sprintf("%s/%s.%s", filepath.Dir(path.Join(provider.MetadataDirectoryName, item.Pathname)), sha.New(item.Name), extension)
}

func typeOfItem(item provider.StorageItem) model.ItemType {
	itemType := model.TypeVideo
	if item.IsImage() {
		itemType = model.TypeImage
	} else if item.IsPdf() {
		itemType = model.TypePDF
	}

	return itemType
}
