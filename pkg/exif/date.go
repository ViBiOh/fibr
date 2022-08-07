package exif

import (
	"context"
	"fmt"

	absto "github.com/ViBiOh/absto/pkg/model"
	exas "github.com/ViBiOh/exas/pkg/model"
)

func (a App) updateDate(ctx context.Context, item absto.Item, data exas.Exif) error {
	if data.Date.IsZero() || item.Date.Equal(data.Date) {
		return nil
	}

	if err := a.storageApp.UpdateDate(ctx, item.Pathname, data.Date); err != nil {
		return fmt.Errorf("update date: %s", err)
	}

	return nil
}
