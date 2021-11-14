package exif

import (
	"fmt"

	"github.com/ViBiOh/exas/pkg/model"
	"github.com/ViBiOh/fibr/pkg/provider"
)

func (a App) updateDate(item provider.StorageItem, data model.Exif) error {
	if data.IsZero() {
		var err error
		if data, err = a.get(item); err != nil {
			return fmt.Errorf("unable to get exif: %s", err)
		}
	}

	if data.Date.IsZero() {
		return nil
	}

	if item.Date.Equal(data.Date) {
		return nil
	}

	if err := a.storageApp.UpdateDate(item.Pathname, data.Date); err != nil {
		return fmt.Errorf("unable to update date: %s", err)
	}

	return nil
}
