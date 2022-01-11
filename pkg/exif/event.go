package exif

import (
	"fmt"

	absto "github.com/ViBiOh/absto/pkg/model"
	exas "github.com/ViBiOh/exas/pkg/model"
	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/httputils/v4/pkg/logger"
)

// EventConsumer handle event pushed to the event bus
func (a App) EventConsumer(e provider.Event) {
	if !a.enabled() {
		return
	}

	switch e.Type {
	case provider.StartEvent:
		if err := a.handleStartEvent(e.Item); err != nil {
			getEventLogger(e.Item).Error("unable to start: %s", err)
		}
	case provider.UploadEvent:
		if err := a.handleUploadEvent(e.Item); err != nil {
			getEventLogger(e.Item).Error("unable to upload: %s", err)
		}
	case provider.RenameEvent:
		if err := a.rename(e.Item, *e.New); err != nil {
			getEventLogger(e.Item).Error("unable to rename: %s", err)
		}
	case provider.DeleteEvent:
		if err := a.delete(e.Item); err != nil {
			getEventLogger(e.Item).Error("unable to delete: %s", err)
		}
	}
}

func getEventLogger(item absto.Item) logger.Provider {
	return logger.WithField("fn", "exif.EventConsumer").WithField("item", item.Pathname)
}

func (a App) handleStartEvent(item absto.Item) error {
	if a.hasMetadata(item) {
		return nil
	}

	if item.IsDir {
		if len(item.Pathname) != 0 {
			return a.aggregate(item)
		}

		return nil
	}

	return a.handleUploadEvent(item)
}

func (a App) handleUploadEvent(item absto.Item) error {
	if !a.CanHaveExif(item) {
		return nil
	}

	if a.amqpClient != nil {
		return a.publishExifRequest(item)
	}

	exif, err := a.extractAndSaveExif(item)
	if err != nil {
		return fmt.Errorf("unable to extract and save exif: %s", err)
	}

	if exif.IsZero() {
		return nil
	}

	return a.processExif(item, exif)
}

func (a App) processExif(item absto.Item, exif exas.Exif) error {
	if err := a.updateDate(item, exif); err != nil {
		return fmt.Errorf("unable to update date: %s", err)
	}

	if err := a.aggregate(item); err != nil {
		return fmt.Errorf("unable to aggregate folder: %s", err)
	}

	return nil
}

func (a App) rename(old, new absto.Item) error {
	oldPath := getExifPath(old)
	if _, err := a.storageApp.Info(oldPath); provider.IsNotExist(err) {
		return nil
	}

	if err := a.storageApp.Rename(oldPath, getExifPath(new)); err != nil {
		return fmt.Errorf("unable to rename exif: %s", err)
	}

	if !old.IsDir {
		if err := a.aggregateOnRename(old, new); err != nil {
			return fmt.Errorf("unable to aggregate on rename: %s", err)
		}
	}

	return nil
}

func (a App) aggregateOnRename(old, new absto.Item) error {
	oldDir, err := a.getDirOf(old)
	if err != nil {
		return fmt.Errorf("unable to get old directory: %s", err)
	}

	newDir, err := a.getDirOf(new)
	if err != nil {
		return fmt.Errorf("unable to get new directory: %s", err)
	}

	if oldDir.Pathname == newDir.Pathname {
		return nil
	}

	if err = a.aggregate(oldDir); err != nil {
		return fmt.Errorf("unable to aggregate old directory: %s", err)
	}

	if err = a.aggregate(newDir); err != nil {
		return fmt.Errorf("unable to aggregate new directory: %s", err)
	}

	return nil
}

func (a App) delete(item absto.Item) error {
	if err := a.storageApp.Remove(getExifPath(item)); err != nil {
		return fmt.Errorf("unable to delete: %s", err)
	}

	if !item.IsDir {
		if err := a.aggregate(item); err != nil {
			return fmt.Errorf("unable to aggregate directory: %s", err)
		}
	}

	return nil
}
