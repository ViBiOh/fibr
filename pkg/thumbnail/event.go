package thumbnail

import (
	"context"
	"fmt"

	absto "github.com/ViBiOh/absto/pkg/model"
	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/httputils/v4/pkg/cache"
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
		oldFilename := a.PathForScale(old, size)

		if err := a.storageApp.Rename(ctx, oldFilename, a.PathForScale(new, size)); err != nil && !absto.IsNotExist(err) {
			return fmt.Errorf("rename thumbnail: %w", err)
		}

		if err := a.redisClient.Delete(ctx, redisKey(oldFilename)); err != nil {
			logger.Error("delete cache: %s", err)
		}

		if provider.VideoExtensions[old.Extension] != "" && a.HasStream(ctx, old) {
			if err := a.renameStream(ctx, old, new); err != nil {
				return fmt.Errorf("rename stream: %w", err)
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
		cacheKey := redisKey(a.PathForScale(event.Item, size))

		if event.GetMetadata("force") == "cache" {
			if err := a.redisClient.Delete(ctx, cacheKey); err != nil {
				logger.WithField("fn", "thumbnail.generate").WithField("item", event.Item.Pathname).Error("flush cache for scale %d: %s", size, err)
			}

			if !forced {
				continue
			}
		}

		if !forced && a.HasThumbnail(ctx, event.Item, size) {
			continue
		}

		if err := cache.EvictOnSuccess(ctx, a.redisClient, cacheKey, a.generate(ctx, event.Item, size)); err != nil {
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
		if err = cache.EvictOnSuccess(ctx, a.redisClient, redisKey(getStreamPath(event.Item)), a.generateStream(ctx, event.Item)); err != nil {
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
		filename := a.PathForScale(item, size)

		if err := a.storageApp.Remove(ctx, filename); err != nil {
			logger.Error("delete thumbnail: %s", err)
		}

		if err := a.redisClient.Delete(ctx, redisKey(filename)); err != nil {
			logger.Error("delete cache: %s", err)
		}

		if provider.VideoExtensions[item.Extension] != "" && a.HasStream(ctx, item) {
			if err := a.deleteStream(ctx, item); err != nil {
				logger.Error("delete stream: %s", err)
			}
		}
	}
}
