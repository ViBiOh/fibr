package thumbnail

import (
	"context"
	"fmt"
	"log/slog"

	absto "github.com/ViBiOh/absto/pkg/model"
	"github.com/ViBiOh/fibr/pkg/provider"
)

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
		if !e.Item.IsDir() {
			if err := a.Rename(ctx, e.Item, *e.New); err != nil {
				slog.Error("rename item", "err", err)
			}
		}
	case provider.DeleteEvent:
		a.delete(ctx, e.Item)
	}
}

func (a App) Rename(ctx context.Context, old, new absto.Item) error {
	if old.IsDir() {
		return nil
	}

	for _, size := range a.sizes {
		oldFilename := a.PathForScale(old, size)

		if err := a.storageApp.Rename(ctx, oldFilename, a.PathForScale(new, size)); err != nil && !absto.IsNotExist(err) {
			return fmt.Errorf("rename thumbnail: %w", err)
		}

		if err := a.redisClient.Delete(ctx, redisKey(oldFilename)); err != nil {
			slog.Error("delete cache", "err", err)
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
		if event.GetMetadata("force") == "cache" {
			if err := a.redisClient.Delete(ctx, redisKey(a.PathForScale(event.Item, size))); err != nil {
				slog.Error("flush cache for scale", "err", err, "fn", "thumbnail.generate", "scale", size, "item", event.Item.Pathname)
			}

			if !forced {
				continue
			}
		}

		if !forced && a.HasThumbnail(ctx, event.Item, size) {
			continue
		}

		if err := a.cacheApp.EvictOnSuccess(ctx, a.PathForScale(event.Item, size), a.generate(ctx, event.Item, size)); err != nil {
			slog.Error("generate for scale %d: %s", "err", err, "scale", size, "item", event.Item.Pathname, "fn", "thumbnail.generate")
		}
	}

	if provider.VideoExtensions[event.Item.Extension] != "" && (forced || !a.HasStream(ctx, event.Item)) {
		a.generateStreamIfNeeded(ctx, event)
	}
}

func (a App) generateStreamIfNeeded(ctx context.Context, event provider.Event) {
	if needStream, err := a.shouldGenerateStream(ctx, event.Item); err != nil {
		slog.Error("determine if stream generation is possible", "err", err)
	} else if needStream {
		if err = a.cacheApp.EvictOnSuccess(ctx, getStreamPath(event.Item), a.generateStream(ctx, event.Item)); err != nil {
			slog.Error("generate stream", "err", err)
		}
	}
}

func (a App) delete(ctx context.Context, item absto.Item) {
	if item.IsDir() {
		if err := a.storageApp.RemoveAll(ctx, provider.MetadataDirectory(item)); err != nil {
			slog.Error("delete thumbnail folder", "err", err)
		}
		return
	}

	for _, size := range a.sizes {
		filename := a.PathForScale(item, size)

		if err := a.storageApp.RemoveAll(ctx, filename); err != nil {
			slog.Error("delete thumbnail", "err", err)
		}

		if err := a.redisClient.Delete(ctx, redisKey(filename)); err != nil {
			slog.Error("delete cache", "err", err)
		}

		if provider.VideoExtensions[item.Extension] != "" && a.HasStream(ctx, item) {
			if err := a.deleteStream(ctx, item); err != nil {
				slog.Error("delete stream", "err", err)
			}
		}
	}
}
