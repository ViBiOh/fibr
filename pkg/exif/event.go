package exif

import (
	"context"
	"fmt"

	absto "github.com/ViBiOh/absto/pkg/model"
	exas "github.com/ViBiOh/exas/pkg/model"
	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/httputils/v4/pkg/logger"
)

// EventConsumer handle event pushed to the event bus
func (a App) EventConsumer(ctx context.Context, e provider.Event) {
	if !a.enabled() {
		return
	}

	var err error

	switch e.Type {
	case provider.StartEvent:
		if err = a.handleStartEvent(ctx, e); err != nil {
			getEventLogger(e.Item).Error("unable to start: %s", err)
		}
	case provider.UploadEvent:
		if err = a.handleUploadEvent(ctx, e.Item, true); err != nil {
			getEventLogger(e.Item).Error("unable to upload: %s", err)
		}
	case provider.RenameEvent:
		if !e.Item.IsDir {
			err = a.Rename(ctx, e.Item, *e.New)
			if err == nil {
				err = a.aggregateOnRename(ctx, e.Item, *e.New)
			}
		}

		if err != nil {
			getEventLogger(e.Item).Error("unable to rename: %s", err)
		}
	case provider.DeleteEvent:
		if err := a.delete(ctx, e.Item); err != nil {
			getEventLogger(e.Item).Error("unable to delete: %s", err)
		}
	}
}

// Rename exif of an item
func (a App) Rename(ctx context.Context, old, new absto.Item) error {
	if err := a.storageApp.Rename(ctx, Path(old), Path(new)); err != nil && !absto.IsNotExist(err) {
		return fmt.Errorf("unable to rename exif: %s", err)
	}

	return nil
}

func getEventLogger(item absto.Item) logger.Provider {
	return logger.WithField("fn", "exif.EventConsumer").WithField("item", item.Pathname)
}

func (a App) handleStartEvent(ctx context.Context, event provider.Event) error {
	forced := event.IsForcedFor("exif")

	item := event.Item
	if !forced && a.hasMetadata(ctx, item) {
		return nil
	}

	if item.IsDir {
		if len(item.Pathname) != 0 {
			return a.aggregate(ctx, item)
		}

		return nil
	}

	return a.handleUploadEvent(ctx, item, !forced)
}

func (a App) handleUploadEvent(ctx context.Context, item absto.Item, aggregate bool) error {
	if !a.CanHaveExif(item) {
		return nil
	}

	if a.amqpClient != nil {
		return a.publishExifRequest(item)
	}

	exif, err := a.extractAndSaveExif(ctx, item)
	if err != nil {
		return fmt.Errorf("unable to extract and save exif: %s", err)
	}

	if exif.IsZero() {
		return nil
	}

	return a.processExif(ctx, item, exif, aggregate)
}

func (a App) processExif(ctx context.Context, item absto.Item, exif exas.Exif, aggregate bool) error {
	if err := a.updateDate(ctx, item, exif); err != nil {
		return fmt.Errorf("unable to update date: %s", err)
	}

	if !aggregate {
		return nil
	}

	if err := a.aggregate(ctx, item); err != nil {
		return fmt.Errorf("unable to aggregate folder: %s", err)
	}

	return nil
}

func (a App) aggregateOnRename(ctx context.Context, old, new absto.Item) error {
	oldDir, err := a.getDirOf(ctx, old)
	if err != nil {
		return fmt.Errorf("unable to get old directory: %s", err)
	}

	newDir, err := a.getDirOf(ctx, new)
	if err != nil {
		return fmt.Errorf("unable to get new directory: %s", err)
	}

	if oldDir.Pathname == newDir.Pathname {
		return nil
	}

	if err = a.aggregate(ctx, oldDir); err != nil {
		return fmt.Errorf("unable to aggregate old directory: %s", err)
	}

	if err = a.aggregate(ctx, newDir); err != nil {
		return fmt.Errorf("unable to aggregate new directory: %s", err)
	}

	return nil
}

func (a App) delete(ctx context.Context, item absto.Item) error {
	if err := a.storageApp.Remove(ctx, Path(item)); err != nil {
		return fmt.Errorf("unable to delete: %s", err)
	}

	if !item.IsDir {
		if err := a.aggregate(ctx, item); err != nil {
			return fmt.Errorf("unable to aggregate directory: %s", err)
		}
	}

	return nil
}
