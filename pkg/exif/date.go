package exif

import (
	"fmt"

	absto "github.com/ViBiOh/absto/pkg/model"
	exas "github.com/ViBiOh/exas/pkg/model"
)

func (a App) updateDate(item absto.Item, data exas.Exif) error {
	if data.Date.IsZero() || item.Date.Equal(data.Date) {
		return nil
	}

	if err := a.storageApp.UpdateDate(item.Pathname, data.Date); err != nil {
		return fmt.Errorf("unable to update date: %s", err)
	}

	return nil
}
