package thumbnail

import (
	"context"
	"fmt"
	"log/slog"

	absto "github.com/ViBiOh/absto/pkg/model"
	"github.com/ViBiOh/fibr/pkg/provider"
)

func (s Service) EventConsumer(ctx context.Context, e provider.Event) {
	if s.vithRequest.IsZero() && s.amqpClient == nil {
		return
	}

	switch e.Type {
	case provider.StartEvent:
		fallthrough
	case provider.UploadEvent:
		s.generateItem(ctx, e)
	case provider.RenameEvent:
		if e.Item.IsDir() {
			// Dir are handled on the event bus
			return
		}

		if err := s.Rename(ctx, e.Item, *e.New); err != nil {
			slog.LogAttrs(ctx, slog.LevelError, "rename item", slog.Any("error", err))
		}
	case provider.DeleteEvent:
		s.delete(ctx, e.Item)
	}
}

func (s Service) Rename(ctx context.Context, old, new absto.Item) error {
	if old.IsDir() {
		return nil
	}

	for _, size := range s.sizes {
		oldFilename := s.PathForScale(old, size)

		if err := s.storage.Rename(ctx, oldFilename, s.PathForScale(new, size)); err != nil && !absto.IsNotExist(err) {
			return fmt.Errorf("rename thumbnail: %w", err)
		}

		if err := s.redisClient.Delete(ctx, redisKey(oldFilename)); err != nil {
			slog.LogAttrs(ctx, slog.LevelError, "delete cache", slog.Any("error", err))
		}

		if provider.VideoExtensions[old.Extension] != "" && s.HasStream(ctx, old) {
			if err := s.renameStream(ctx, old, new); err != nil {
				return fmt.Errorf("rename stream: %w", err)
			}
		}
	}

	return nil
}

func (s Service) generateItem(ctx context.Context, event provider.Event) {
	if !s.CanGenerateThumbnail(event.Item) {
		return
	}

	forced := event.IsForcedFor("thumbnail")

	for _, size := range s.sizes {
		if event.GetMetadata("force") == "cache" {
			if err := s.redisClient.Delete(ctx, redisKey(s.PathForScale(event.Item, size))); err != nil {
				slog.LogAttrs(ctx, slog.LevelError, "flush cache for scale", slog.String("fn", "thumbnail.generate"), slog.Uint64("scale", size), slog.String("item", event.Item.Pathname), slog.Any("error", err))
			}

			if !forced {
				continue
			}
		}

		if !forced && s.HasThumbnail(ctx, event.Item, size) {
			continue
		}

		if err := s.cache.EvictOnSuccess(ctx, s.PathForScale(event.Item, size), s.generate(ctx, event.Item, size)); err != nil {
			slog.LogAttrs(ctx, slog.LevelError, "generate for scale", slog.String("fn", "thumbnail.generate"), slog.Uint64("scale", size), slog.String("item", event.Item.Pathname), slog.Any("error", err))
		}
	}

	if provider.VideoExtensions[event.Item.Extension] != "" && (forced || !s.HasStream(ctx, event.Item)) {
		s.generateStreamIfNeeded(ctx, event)
	}
}

func (s Service) generateStreamIfNeeded(ctx context.Context, event provider.Event) {
	if needStream, err := s.shouldGenerateStream(ctx, event.Item); err != nil {
		slog.LogAttrs(ctx, slog.LevelError, "determine if stream generation is possible", slog.Any("error", err))
	} else if needStream {
		if err = s.cache.EvictOnSuccess(ctx, getStreamPath(event.Item), s.generateStream(ctx, event.Item)); err != nil {
			slog.LogAttrs(ctx, slog.LevelError, "generate stream", slog.Any("error", err))
		}
	}
}

func (s Service) delete(ctx context.Context, item absto.Item) {
	if item.IsDir() {
		if err := s.storage.RemoveAll(ctx, provider.MetadataDirectory(item)); err != nil {
			slog.LogAttrs(ctx, slog.LevelError, "delete thumbnail folder", slog.Any("error", err))
		}
		return
	}

	for _, size := range s.sizes {
		filename := s.PathForScale(item, size)

		if err := s.storage.RemoveAll(ctx, filename); err != nil {
			slog.LogAttrs(ctx, slog.LevelError, "delete thumbnail", slog.Any("error", err))
		}

		if err := s.redisClient.Delete(ctx, redisKey(filename)); err != nil {
			slog.LogAttrs(ctx, slog.LevelError, "delete cache", slog.Any("error", err))
		}

		if provider.VideoExtensions[item.Extension] != "" && s.HasStream(ctx, item) {
			if err := s.deleteStream(ctx, item); err != nil {
				slog.LogAttrs(ctx, slog.LevelError, "delete stream", slog.Any("error", err))
			}
		}
	}
}
