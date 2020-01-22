package crud

import (
	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/fibr/pkg/thumbnail"
	"github.com/ViBiOh/httputils/v3/pkg/logger"
)

func (a *app) createThumbnail(item provider.StorageItem) {
	if item.IsDir || thumbnail.CanHaveThumbnail(item) {
		a.thumbnail.AsyncGenerateThumbnail(item)
	}
}

func (a *app) deleteThumbnail(item provider.StorageItem) bool {
	thumbnailPath, ok := a.thumbnail.HasThumbnail(item)
	if !ok {
		return false
	}

	if err := a.storage.Remove(thumbnailPath); err != nil {
		logger.Error("unable to delete thumbnail: %s", err)
	}

	return true
}

func (a *app) renameThumbnail(oldItem, newItem provider.StorageItem) {
	if !a.deleteThumbnail(oldItem) {
		return
	}

	a.createThumbnail(newItem)
}
