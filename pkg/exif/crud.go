package exif

import (
	"fmt"

	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/httputils/v4/pkg/logger"
)

func (a app) rename(old, new provider.StorageItem) error {
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
		oldDir, err := a.getDirOf(old)
		if err != nil {
			return fmt.Errorf("unable to get old directory: %s", err)
		}

		newDir, err := a.getDirOf(new)
		if err != nil {
			return fmt.Errorf("unable to get new directory: %s", err)
		}

		if oldDir.Pathname != newDir.Pathname {
			if err := a.aggregate(oldDir); err != nil {
				return fmt.Errorf("unable to aggregate old directory: %s", err)
			}

			if err := a.aggregate(newDir); err != nil {
				return fmt.Errorf("unable to aggregate new directory: %s", err)
			}
		}
	}

	return nil
}

func (a app) delete(item provider.StorageItem) error {
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

func (a app) EventConsumer(e provider.Event) {
	if !a.enabled() {
		return
	}

	switch e.Type {
	case provider.StartEvent:
		if CanHaveExif(e.Item) {
			if !a.HasExif(e.Item) {
				if _, err := a.get(e.Item); err != nil {
					logger.Error("unable to get exif for `%s`: %s", e.Item.Pathname, err)
				}
			}

			if !a.HasGeocode(e.Item) {
				a.geocode(e.Item)
			}

			if a.dateOnStart {
				if err := a.updateDate(e.Item); err != nil {
					logger.Error("unable to update date for `%s`: %s", e.Item.Pathname, err)
				}
			}
		}

		if e.Item.IsDir && a.aggregateOnStart {
			if err := a.aggregate(e.Item); err != nil {
				logger.Error("unable to aggregate exif for `%s`: %s", e.Item.Pathname, err)
			}
		}
	case provider.UploadEvent:
		if !CanHaveExif(e.Item) {
			return
		}

		if err := a.updateDate(e.Item); err != nil {
			logger.Error("unable to update date for `%s`: %s", e.Item.Pathname, err)
		}

		a.geocode(e.Item)

		if err := a.aggregate(e.Item); err != nil {
			logger.Error("unable to aggregate exif for `%s`: %s", e.Item.Pathname, err)
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
