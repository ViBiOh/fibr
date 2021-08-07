package thumbnail

import (
	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/httputils/v4/pkg/logger"
)

func (a App) rename(old, new provider.StorageItem) {
	oldPath := getThumbnailPath(old)
	if _, err := a.storageApp.Info(oldPath); provider.IsNotExist(err) {
		return
	}

	if err := a.storageApp.Rename(oldPath, getThumbnailPath(new)); err != nil {
		logger.Error("unable to rename thumbnail: %s", err)
	}
}

func (a App) delete(item provider.StorageItem) {
	if err := a.storageApp.Remove(getThumbnailPath(item)); err != nil {
		logger.Error("unable to delete thumbnail: %s", err)
	}
}

// EventConsumer handle event pushed to the event bus
func (a App) EventConsumer(e provider.Event) {
	if !a.enabled() {
		return
	}

	switch e.Type {
	case provider.StartEvent:
		fallthrough
	case provider.UploadEvent:
		if CanHaveThumbnail(e.Item) && !a.HasThumbnail(e.Item) {
			if err := a.generate(e.Item); err != nil {
				logger.Error("unable to generate thumbnail for `%s`: %s", e.Item.Pathname, err)
			}
		}
	case provider.RenameEvent:
		a.rename(e.Item, e.New)
	case provider.DeleteEvent:
		a.delete(e.Item)
	}
}
