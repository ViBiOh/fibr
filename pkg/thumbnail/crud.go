package thumbnail

import (
	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/httputils/v3/pkg/logger"
)

// Remove thumbnail of given item
func (a app) Remove(item provider.StorageItem) {
	if !a.Enabled() {
		return
	}

	if err := a.storage.Remove(getThumbnailPath(item)); err != nil {
		logger.Error("%s", err)
	}
}

// Rename thumbnails of given items
func (a app) Rename(old, new provider.StorageItem) {
	if !a.Enabled() {
		return
	}

	if err := a.storage.Rename(getThumbnailPath(old), getThumbnailPath(new)); err != nil {
		logger.Error("%s", err)
	}
}
