package thumbnail

import (
	"context"

	absto "github.com/ViBiOh/absto/pkg/model"
	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/httputils/v4/pkg/logger"
)

// EventConsumer handle event pushed to the event bus
func (a App) EventConsumer(e provider.Event) {
	if a.vithRequest.IsZero() {
		return
	}

	switch e.Type {
	case provider.StartEvent:
		fallthrough
	case provider.UploadEvent:
		a.generateItem(e.Item)
	case provider.RenameEvent:
		a.rename(e.Item, *e.New)
	case provider.DeleteEvent:
		a.delete(e.Item)
	}
}

func (a App) generateItem(item absto.Item) {
	if !a.CanHaveThumbnail(item) {
		return
	}

	if !a.HasThumbnail(item) {
		if err := a.generate(item); err != nil {
			logger.WithField("fn", "thumbnail.generate").WithField("item", item.Pathname).Error("unable to generate: %s", err)
		}
	}

	if item.IsVideo() && !a.HasStream(item) {
		if needStream, err := a.shouldGenerateStream(context.Background(), item); err != nil {
			logger.Error("unable to determine if stream generation is possible: %s", err)
		} else if needStream {
			if err := a.generateStream(context.Background(), item); err != nil {
				logger.Error("unable to generate stream: %s", err)
			}
		}
	}
}

func (a App) rename(old, new absto.Item) {
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

func (a App) delete(item absto.Item) {
	if err := a.storageApp.Remove(getThumbnailPath(item)); err != nil {
		logger.Error("unable to delete thumbnail: %s", err)
	}

	if item.IsVideo() && a.HasStream(item) {
		if err := a.deleteStream(context.Background(), item); err != nil {
			logger.Error("unable to delete stream: %s", err)
		}
	}
}
