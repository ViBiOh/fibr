package thumbnail

import (
	"context"

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

	if old.IsVideo() && a.HasStream(old) {
		if err := a.renameStream(context.Background(), old, new); err != nil {
			logger.Error("unable to rename stream: %s", err)
		}
	}
}

func (a App) delete(item provider.StorageItem) {
	if err := a.storageApp.Remove(getThumbnailPath(item)); err != nil {
		logger.Error("unable to delete thumbnail: %s", err)
	}

	if item.IsVideo() && a.HasStream(item) {
		if err := a.deleteStream(context.Background(), item); err != nil {
			logger.Error("unable to delete stream: %s", err)
		}
	}
}

// EventConsumer handle event pushed to the event bus
func (a App) EventConsumer(e provider.Event) {
	if !a.videoEnabled() && !a.imageEnabled() {
		return
	}

	switch e.Type {
	case provider.StartEvent:
		fallthrough
	case provider.UploadEvent:
		if !a.CanHaveThumbnail(e.Item) {
			return
		}

		if (e.Item.IsVideo() && !a.videoEnabled()) || (e.Item.IsImage() && !a.imageEnabled()) {
			return
		}

		if !a.HasThumbnail(e.Item) {
			if err := a.generate(e.Item); err != nil {
				logger.Error("unable to generate thumbnail for `%s`: %s", e.Item.Pathname, err)
			}
		}

		if e.Item.IsVideo() && !a.HasStream(e.Item) {
			needStream, err := a.shouldGenerateStream(context.Background(), e.Item)
			if err != nil {
				logger.Error("unable to determine if stream generation is possible: %s", err)
			} else if needStream {
				if err := a.generateStream(context.Background(), e.Item); err != nil {
					logger.Error("unable to generate stream: %s", err)
				}
			}
		}

	case provider.RenameEvent:
		a.rename(e.Item, *e.New)
	case provider.DeleteEvent:
		a.delete(e.Item)
	}
}
