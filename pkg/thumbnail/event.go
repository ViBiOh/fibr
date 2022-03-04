package thumbnail

import (
	"context"

	absto "github.com/ViBiOh/absto/pkg/model"
	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/httputils/v4/pkg/logger"
)

// EventConsumer handle event pushed to the event bus
func (a App) EventConsumer(ctx context.Context, e provider.Event) {
	if a.vithRequest.IsZero() {
		return
	}

	switch e.Type {
	case provider.StartEvent:
		fallthrough
	case provider.UploadEvent:
		a.generateItem(ctx, e)
	case provider.RenameEvent:
		a.rename(ctx, e.Item, *e.New)
	case provider.DeleteEvent:
		a.delete(ctx, e.Item)
	}
}

func (a App) generateItem(ctx context.Context, event provider.Event) {
	if !a.CanHaveThumbnail(event.Item) {
		return
	}

	if event.GetMetadata("force") == "true" || !a.HasThumbnail(event.Item) {
		if err := a.generate(ctx, event.Item, SmallSize); err != nil {
			logger.WithField("fn", "thumbnail.generate").WithField("item", event.Item.Pathname).Error("unable to generate for scale %d: %s", SmallSize, err)
		}
	}

	if provider.VideoExtensions[event.Item.Extension] != "" && (event.GetMetadata("force") == "true" || !a.HasStream(event.Item)) {
		if needStream, err := a.shouldGenerateStream(ctx, event.Item); err != nil {
			logger.Error("unable to determine if stream generation is possible: %s", err)
		} else if needStream {
			if err := a.generateStream(ctx, event.Item); err != nil {
				logger.Error("unable to generate stream: %s", err)
			}
		}
	}
}

func (a App) rename(ctx context.Context, old, new absto.Item) {
	oldPath := getThumbnailPath(old)
	if _, err := a.storageApp.Info(oldPath); absto.IsNotExist(err) {
		return
	}

	if err := a.storageApp.Rename(oldPath, getThumbnailPath(new)); err != nil {
		logger.Error("unable to rename thumbnail: %s", err)
	}

	if provider.VideoExtensions[old.Extension] != "" && a.HasStream(old) {
		if err := a.renameStream(ctx, old, new); err != nil {
			logger.Error("unable to rename stream: %s", err)
		}
	}
}

func (a App) delete(ctx context.Context, item absto.Item) {
	if err := a.storageApp.Remove(getThumbnailPath(item)); err != nil {
		logger.Error("unable to delete thumbnail: %s", err)
	}

	if provider.VideoExtensions[item.Extension] != "" && a.HasStream(item) {
		if err := a.deleteStream(ctx, item); err != nil {
			logger.Error("unable to delete stream: %s", err)
		}
	}
}
