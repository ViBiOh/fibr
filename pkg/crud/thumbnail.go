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

func (a *app) deleteThumbnail(item provider.StorageItem) {
	thumbnailPath, ok := a.thumbnail.HasThumbnail(item)
	if !ok {
		return
	}

	if err := a.storage.Remove(thumbnailPath); err != nil {
		logger.Error("unable to delete thumbnail: %s", err)
	}
}

func (a *app) renameThumbnail(oldItem, newItem provider.StorageItem) {
	a.deleteThumbnail(oldItem)
	a.createThumbnail(newItem)
}
