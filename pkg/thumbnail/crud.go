package thumbnail

import (
	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/httputils/v4/pkg/logger"
)

func (a app) Delete(item provider.StorageItem) {
	if !a.Enabled() {
		return
	}

	if err := a.storage.Remove(getThumbnailPath(item)); err != nil {
		logger.Error("%s", err)
	}
}

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
