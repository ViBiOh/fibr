package metadata

import (
	"context"
	"fmt"

	absto "github.com/ViBiOh/absto/pkg/model"
	"github.com/ViBiOh/fibr/pkg/provider"
)

func (a App) updateDate(ctx context.Context, item absto.Item, data provider.Metadata) error {
	if data.Date.IsZero() || item.Date.Equal(data.Date) {
		return nil
	}

	if err := a.storageApp.UpdateDate(ctx, item.Pathname, data.Date); err != nil {
		return fmt.Errorf("update date: %w", err)
	}

	return nil
}
