package exif

import (
	"fmt"

	"github.com/ViBiOh/exas/pkg/model"
	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/httputils/v4/pkg/logger"
)

var eventConsumerLogger = logger.WithField("fn", "exif.EventConsumer")

// EventConsumer handle event pushed to the event bus
func (a App) EventConsumer(e provider.Event) {
	if !a.enabled() {
		return
	}

	switch e.Type {
	case provider.StartEvent:
		if err := a.handleStartEvent(e.Item); err != nil {
			eventConsumerLogger.WithField("item", e.Item.Pathname).Error("unable to start: %s", err)
		}
	case provider.UploadEvent:
		if err := a.handleUploadEvent(e.Item); err != nil {
			eventConsumerLogger.WithField("item", e.Item.Pathname).Error("unable to upload: %s", err)
		}
	case provider.RenameEvent:
		if err := a.rename(e.Item, *e.New); err != nil {
			eventConsumerLogger.WithField("item", e.Item.Pathname).Error("unable to rename: %s", err)
		}
	case provider.DeleteEvent:
		if err := a.delete(e.Item); err != nil {
			eventConsumerLogger.WithField("item", e.Item.Pathname).Error("unable to delete: %s", err)
		}
	}
}

func (a App) handleStartEvent(item provider.StorageItem) error {
	if a.hasMetadata(item) {
		return nil
	}

	if item.IsDir {
		return a.aggregate(item)
	}
	
	if !a.CanHaveExif(item) {
		return nil
	}

	if a.amqpClient != nil {
		return a.askForExif(item)
	}

	data, err := a.get(item)
	if err != nil {
		return fmt.Errorf("unable to get exif : %s", err)
	}

	return a.processExif(item, data)
}

func (a App) handleUploadEvent(item provider.StorageItem) error {
	if !a.CanHaveExif(item) {
		return nil
	}

	if a.amqpClient != nil {
		return a.askForExif(item)
	}

	data, err := a.get(item)
	if err != nil {
		return fmt.Errorf("unable to get exif: %s", err)
	}

	return a.processExif(item, data)
}

func (a App) processExif(item provider.StorageItem, exif model.Exif) error {
	if err := a.updateDate(item, exif); err != nil {
		return fmt.Errorf("unable to update date: %s", err)
	}

	if err := a.aggregate(item); err != nil {
		return fmt.Errorf("unable to aggregate folder: %s", err)
	}

	return nil
}

func (a App) rename(old, new provider.StorageItem) error {
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

func (a App) aggregateOnRename(old, new provider.StorageItem) error {
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

func (a App) delete(item provider.StorageItem) error {
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
