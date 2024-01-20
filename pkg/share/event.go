package share

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	absto "github.com/ViBiOh/absto/pkg/model"
	"github.com/ViBiOh/fibr/pkg/exclusive"
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
	_, err := s.Exclusive(ctx, old.ID, exclusive.Duration, func(ctx context.Context) error {
		for id, share := range s.shares {
			if strings.HasPrefix(share.Path, old.Pathname) {
				share.Path = strings.Replace(share.Path, old.Pathname, new.Pathname, 1)
				s.shares[id] = share

				if err := s.redisClient.PublishJSON(ctx, s.pubsubChannel, share); err != nil {
					return fmt.Errorf("publish share rename: %w", err)
				}
			}
		}

		return provider.SaveJSON(ctx, s.storage, shareFilename, s.shares)
	})

	return err
}

func (s *Service) deleteItem(ctx context.Context, item absto.Item) error {
	_, err := s.Exclusive(ctx, item.ID, exclusive.Duration, func(_ context.Context) error {
		for id, share := range s.shares {
			if strings.HasPrefix(share.Path, item.Pathname) {
				if err := s.delete(ctx, id); err != nil {
					return fmt.Errorf("delete share `%s`: %w", id, err)
				}
			}
		}

		return nil
	})

	return err
}
