package thumbnail

import (
	"context"
	"fmt"

	absto "github.com/ViBiOh/absto/pkg/model"
	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/httputils/v4/pkg/logger"
)

// EventConsumer handle event pushed to the event bus
func (a App) EventConsumer(ctx context.Context, e provider.Event) {
	if a.vithRequest.IsZero() && a.amqpClient == nil {
		return
	}

	switch e.Type {
	case provider.StartEvent:
		fallthrough
	case provider.UploadEvent:
		a.generateItem(ctx, e)
	case provider.RenameEvent:
		if !e.Item.IsDir {
			if err := a.Rename(ctx, e.Item, *e.New); err != nil {
				logger.Error("rename item: %s", err)
			}
		}
	case provider.DeleteEvent:
		a.delete(ctx, e.Item)
	}
}

// Rename thumbnail of an item
func (a App) Rename(ctx context.Context, old, new absto.Item) error {
	if old.IsDir {
		return nil
	}

	for _, size := range a.sizes {
		if err := a.storageApp.Rename(ctx, a.PathForScale(old, size), a.PathForScale(new, size)); err != nil && !absto.IsNotExist(err) {
			return fmt.Errorf("rename thumbnail: %s", err)
		}

		if provider.VideoExtensions[old.Extension] != "" && a.HasStream(ctx, old) {
			if err := a.renameStream(ctx, old, new); err != nil {
				return fmt.Errorf("rename stream: %s", err)
			}
		}
	}

	return nil
}

func (a App) generateItem(ctx context.Context, event provider.Event) {
	if !a.CanHaveThumbnail(event.Item) {
		return
	}

	forced := event.IsForcedFor("thumbnail")

	for _, size := range a.sizes {
		if !forced && a.HasThumbnail(ctx, event.Item, size) {
			continue
		}

		if err := a.generate(ctx, event.Item, size); err != nil {
			logger.WithField("fn", "thumbnail.generate").WithField("item", event.Item.Pathname).Error("generate for scale %d: %s", size, err)
		}
	}

	if provider.VideoExtensions[event.Item.Extension] != "" && (forced || !a.HasStream(ctx, event.Item)) {
		a.generateStreamIfNeeded(ctx, event)
	}
}

func (a App) generateStreamIfNeeded(ctx context.Context, event provider.Event) {
	if needStream, err := a.shouldGenerateStream(ctx, event.Item); err != nil {
		logger.Error("determine if stream generation is possible: %s", err)
	} else if needStream {
		if err = a.generateStream(ctx, event.Item); err != nil {
			logger.Error("generate stream: %s", err)
		}
	}
}

func (a App) delete(ctx context.Context, item absto.Item) {
	if item.IsDir {
		if err := a.storageApp.Remove(ctx, provider.MetadataDirectory(item)); err != nil {
			logger.Error("delete thumbnail folder: %s", err)
		}
		return
	}

	for _, size := range a.sizes {
		if err := a.storageApp.Remove(ctx, a.PathForScale(item, size)); err != nil {
			logger.Error("delete thumbnail: %s", err)
		}

		if provider.VideoExtensions[item.Extension] != "" && a.HasStream(ctx, item) {
			if err := a.deleteStream(ctx, item); err != nil {
				logger.Error("delete stream: %s", err)
			}
		}
	}
}
