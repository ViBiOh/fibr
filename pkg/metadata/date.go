package metadata

import (
	"context"
	"fmt"
	"log/slog"

	absto "github.com/ViBiOh/absto/pkg/model"
	"github.com/ViBiOh/fibr/pkg/provider"
)

func (s *Service) updateDate(ctx context.Context, item absto.Item, data provider.Metadata) error {
	if data.Date.IsZero() || item.Date.Equal(data.Date) {
		slog.LogAttrs(ctx, slog.LevelDebug, "no exif date or already equal", slog.String("item", item.Pathname), slog.Time("item_date", item.Date), slog.Time("exif_date", data.Date))
		return nil
	}

	if err := s.storage.UpdateDate(ctx, item.Pathname, data.Date); err != nil {
		return fmt.Errorf("update date: %w", err)
	}

	return nil
}
