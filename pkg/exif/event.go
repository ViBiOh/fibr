package exif

import (
	"fmt"

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
			logger.Error("unable to start exif for `%s`: %s", e.Item.Pathname, err)
		}
	case provider.UploadEvent:
		if err := a.handleUploadEvent(e.Item); err != nil {
			logger.Error("unable to upload exif for `%s`: %s", e.Item.Pathname, err)
		}
	case provider.RenameEvent:
		if err := a.rename(e.Item, e.New); err != nil {
			logger.Error("unable to rename exif for `%s`: %s", e.Item.Pathname, err)
		}
	case provider.DeleteEvent:
		if err := a.rename(e.Item, e.New); err != nil {
			logger.Error("unable to delete exif for `%s`: %s", e.Item.Pathname, err)
		}
	}
}

func (a App) handleStartEvent(item provider.StorageItem) error {
	if CanHaveExif(item) {
		if !a.hasExif(item) {
			if _, err := a.get(item); err != nil {
				return fmt.Errorf("unable to get exif : %s", err)
			}
		}

		if !a.hasGeocode(item) {
			a.geocode(item)
		}

		if a.dateOnStart {
			if err := a.updateDate(item); err != nil {
				return fmt.Errorf("unable to update date : %s", err)
			}
		}
	}

	if item.IsDir && a.aggregateOnStart {
		if err := a.aggregate(item); err != nil {
			return fmt.Errorf("unable to aggregate exif : %s", err)
		}
	}

	return nil
}

func (a App) handleUploadEvent(item provider.StorageItem) error {
	if !CanHaveExif(item) {
		return nil
	}

	if err := a.updateDate(item); err != nil {
		return fmt.Errorf("unable to update date: %s", err)
	}

	a.geocode(item)

	if err := a.aggregate(item); err != nil {
		return fmt.Errorf("unable to aggregate exif: %s", err)
	}

	return nil
}

func (a App) rename(old, new provider.StorageItem) error {
	for _, suffix := range metadataFilenames {
		oldPath := getExifPath(old, suffix)
		if _, err := a.storageApp.Info(oldPath); provider.IsNotExist(err) {
			return nil
		}

		if err := a.storageApp.Rename(oldPath, getExifPath(new, suffix)); err != nil {
			return fmt.Errorf("unable to rename exif: %s", err)
		}
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

	if err := a.aggregate(oldDir); err != nil {
		return fmt.Errorf("unable to aggregate old directory: %s", err)
	}

	if err := a.aggregate(newDir); err != nil {
		return fmt.Errorf("unable to aggregate new directory: %s", err)
	}

	return nil
}

func (a App) delete(item provider.StorageItem) error {
	for _, suffix := range metadataFilenames {
		if err := a.storageApp.Remove(getExifPath(item, suffix)); err != nil {
			return fmt.Errorf("unable to delete: %s", err)
		}
	}

	if !item.IsDir {
		if err := a.aggregate(item); err != nil {
			return fmt.Errorf("unable to aggregate directory: %s", err)
		}
	}

	return nil
}
