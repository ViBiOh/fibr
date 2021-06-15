package thumbnail

import (
	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/httputils/v4/pkg/logger"
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

	oldPath := getThumbnailPath(old)
	if _, err := a.storage.Info(oldPath); provider.IsNotExist(err) {
		return
	}

	if err := a.storage.Rename(oldPath, getThumbnailPath(new)); err != nil {
		logger.Error("%s", err)
	}
}
