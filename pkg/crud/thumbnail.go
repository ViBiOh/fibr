package crud

import (
	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/fibr/pkg/thumbnail"
	"github.com/ViBiOh/httputils/v3/pkg/logger"
)

func (a *app) renameThumbnail(oldItem, newItem *provider.StorageItem) {
	if !a.deleteThumbnail(oldItem) {
		return
	}

	a.createThumbnail(newItem)
}

func (a *app) createThumbnail(item *provider.StorageItem) {
	if thumbnail.CanHaveThumbnail(item) {
		a.thumbnail.AsyncGenerateThumbnail(item)
	}
}

func (a *app) deleteThumbnail(item *provider.StorageItem) bool {
	thumbnailPath, ok := a.thumbnail.HasThumbnail(item)
	if !ok {
		return false
	}

	if err := a.storage.Remove(thumbnailPath); err != nil {
		logger.Error("%#v", err)
	}

	return true
}
