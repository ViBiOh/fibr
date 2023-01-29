package metadata

import (
	"context"
	"fmt"

	absto "github.com/ViBiOh/absto/pkg/model"
	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/httputils/v4/pkg/logger"
)

func (a App) updateDate(ctx context.Context, item absto.Item, data provider.Metadata) error {
	if data.Date.IsZero() || item.Date.Equal(data.Date) {
		logger.WithField("item", item.Pathname).
			WithField("item_date", item.Date.String()).
			WithField("exif_date", data.Date.String()).
			Debug("no exif date or already equal")
		return nil
	}

	if err := a.storageApp.UpdateDate(ctx, item.Pathname, data.Date); err != nil {
		return fmt.Errorf("update date: %w", err)
	}

	return nil
}
