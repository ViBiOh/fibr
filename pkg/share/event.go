package share

import (
	"context"
	"log/slog"
	"strings"

	absto "github.com/ViBiOh/absto/pkg/model"
	"github.com/ViBiOh/fibr/pkg/provider"
)

func (s *Service) EventConsumer(ctx context.Context, e provider.Event) {
	switch e.Type {
	case provider.RenameEvent:
		if err := s.renameItem(ctx, e.Item, *e.New); err != nil {
			slog.LogAttrs(ctx, slog.LevelError, "rename share", slog.Any("error", err))
		}
	case provider.DeleteEvent:
		if err := s.deleteItem(ctx, e.Item); err != nil {
			slog.LogAttrs(ctx, slog.LevelError, "delete share", slog.Any("error", err))
		}
	}
}

func (s *Service) renameItem(ctx context.Context, old, new absto.Item) error {
	return s.shares.Update(ctx, func(shares map[string]provider.Share) ([]provider.Share, error) {
		var renamed []provider.Share

		for id, share := range shares {
			if strings.HasPrefix(share.Path, old.Pathname) {
				share.Path = strings.Replace(share.Path, old.Pathname, new.Pathname, 1)
				shares[id] = share

				renamed = append(renamed, share)
			}
		}

		return renamed, nil
	})
}

func (s *Service) deleteItem(ctx context.Context, item absto.Item) error {
	return s.shares.Update(ctx, func(shares map[string]provider.Share) ([]provider.Share, error) {
		var deleted []provider.Share

		for id, share := range shares {
			if strings.HasPrefix(share.Path, item.Pathname) {
				delete(shares, id)

				deleted = append(deleted, provider.Share{ID: id})
			}
		}

		return deleted, nil
	})
}
