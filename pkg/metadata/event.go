package metadata

import (
	"context"
	"fmt"
	"log/slog"

	absto "github.com/ViBiOh/absto/pkg/model"
	"github.com/ViBiOh/fibr/pkg/provider"
)

func (s Service) EventConsumer(ctx context.Context, e provider.Event) {
	if !s.enabled() {
		return
	}

	var err error

	switch e.Type {
	case provider.StartEvent:
		if err = s.handleStartEvent(ctx, e); err != nil {
			getEventLogger(e.Item).ErrorContext(ctx, "start", "error", err)
		}
	case provider.UploadEvent:
		if err = s.handleUploadEvent(ctx, e.Item, true); err != nil {
			getEventLogger(e.Item).ErrorContext(ctx, "upload", "error", err)
		}
	case provider.RenameEvent:
		if !e.Item.IsDir() {
			err = s.Rename(ctx, e.Item, *e.New)
			if err == nil {
				err = s.aggregateOnRename(ctx, e.Item, *e.New)
			}
		}

		if err != nil {
			getEventLogger(e.Item).ErrorContext(ctx, "rename", "error", err)
		}
	case provider.DeleteEvent:
		if err := s.delete(ctx, e.Item); err != nil {
			getEventLogger(e.Item).ErrorContext(ctx, "delete", "error", err)
		}
	}
}

func (s Service) Rename(ctx context.Context, old, new absto.Item) error {
	if err := s.storage.Rename(ctx, Path(old), Path(new)); err != nil && !absto.IsNotExist(err) {
		return fmt.Errorf("rename exif: %w", err)
	}

	if err := s.redisClient.Delete(ctx, redisKey(old)); err != nil {
		return fmt.Errorf("cache: %s", err)
	}

	return nil
}

func getEventLogger(item absto.Item) *slog.Logger {
	return slog.With("fn", "exif.EventConsumer").With("item", item.Pathname)
}

func (s Service) handleStartEvent(ctx context.Context, event provider.Event) error {
	forced := event.IsForcedFor("exif")

	if event.GetMetadata("force") == "cache" {
		if err := s.redisClient.Delete(ctx, redisKey(event.Item)); err != nil {
			slog.ErrorContext(ctx, "flush cache", "error", err, "fn", "exif.startEvent", "item", event.Item.Pathname)
		}

		if !forced {
			return nil
		}
	}

	item := event.Item
	if !forced && s.hasMetadata(ctx, item) {
		slog.DebugContext(ctx, "has metadata", "item", item.Pathname)
		return nil
	}

	if item.IsDir() {
		if len(item.Pathname) != 0 {
			return s.aggregate(ctx, item)
		}

		return nil
	}

	return s.handleUploadEvent(ctx, item, false)
}

func (s Service) handleUploadEvent(ctx context.Context, item absto.Item, aggregate bool) error {
	if !s.CanHaveExif(item) {
		slog.DebugContext(ctx, "can't have exif", "item", item.Pathname)
		return nil
	}

	if s.amqpClient != nil {
		return s.publishExifRequest(ctx, item)
	}

	metadata, err := s.extractAndSaveExif(ctx, item)
	if err != nil {
		return fmt.Errorf("extract and save exif: %w", err)
	}

	if metadata.IsZero() {
		return nil
	}

	return s.processMetadata(ctx, item, metadata, aggregate)
}

func (s Service) processMetadata(ctx context.Context, item absto.Item, exif provider.Metadata, aggregate bool) error {
	if err := s.updateDate(ctx, item, exif); err != nil {
		return fmt.Errorf("update date: %w", err)
	}

	if !aggregate {
		return nil
	}

	if err := s.aggregate(ctx, item); err != nil {
		return fmt.Errorf("aggregate folder: %w", err)
	}

	return nil
}

func (s Service) aggregateOnRename(ctx context.Context, old, new absto.Item) error {
	oldDir, err := s.getDirOf(ctx, old)
	if err != nil {
		return fmt.Errorf("get old directory: %w", err)
	}

	newDir, err := s.getDirOf(ctx, new)
	if err != nil {
		return fmt.Errorf("get new directory: %w", err)
	}

	if oldDir.Pathname == newDir.Pathname {
		return nil
	}

	if err = s.aggregate(ctx, oldDir); err != nil {
		return fmt.Errorf("aggregate old directory: %w", err)
	}

	if err = s.aggregate(ctx, newDir); err != nil {
		return fmt.Errorf("aggregate new directory: %w", err)
	}

	return nil
}

func (s Service) delete(ctx context.Context, item absto.Item) error {
	if err := s.storage.RemoveAll(ctx, Path(item)); err != nil {
		return fmt.Errorf("delete: %w", err)
	}

	if err := s.redisClient.Delete(ctx, redisKey(item)); err != nil {
		return fmt.Errorf("cache: %s", err)
	}

	if !item.IsDir() {
		if err := s.aggregate(ctx, item); err != nil {
			return fmt.Errorf("aggregate directory: %w", err)
		}
	}

	return nil
}
