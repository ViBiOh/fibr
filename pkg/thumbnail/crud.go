package thumbnail

import (
	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/httputils/v4/pkg/logger"
)

func (a app) Delete(item provider.StorageItem) {
	if !a.enabled() {
		return
	}

	if err := a.storageApp.Remove(getThumbnailPath(item)); err != nil {
		logger.Error("unable to delete thumbnail: %s", err)
	}
}

func (a app) Rename(old, new provider.StorageItem) {
	if !a.enabled() {
		return
	}

	oldPath := getThumbnailPath(old)
	if _, err := a.storageApp.Info(oldPath); provider.IsNotExist(err) {
		return
	}

	if err := a.storageApp.Rename(oldPath, getThumbnailPath(new)); err != nil {
		logger.Error("unable to rename thumbnail: %s", err)
	}
}
