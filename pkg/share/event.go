package share

import (
	"context"
	"fmt"
	"strings"

	absto "github.com/ViBiOh/absto/pkg/model"
	"github.com/ViBiOh/fibr/pkg/exclusive"
	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/httputils/v4/pkg/logger"
)

func (a *App) EventConsumer(ctx context.Context, e provider.Event) {
	switch e.Type {
	case provider.RenameEvent:
		if err := a.renameItem(ctx, e.Item, *e.New); err != nil {
			logger.Error("rename share: %s", err)
		}
	case provider.DeleteEvent:
		if err := a.deleteItem(ctx, e.Item); err != nil {
			logger.Error("rename share: %s", err)
		}
	}
}

func (a *App) renameItem(ctx context.Context, old, new absto.Item) error {
	_, err := a.Exclusive(ctx, old.ID, exclusive.Duration, func(ctx context.Context) error {
		for id, share := range a.shares {
			if strings.HasPrefix(share.Path, old.Pathname) {
				share.Path = strings.Replace(share.Path, old.Pathname, new.Pathname, 1)
				a.shares[id] = share

				if err := a.redisClient.PublishJSON(ctx, a.pubsubChannel, share); err != nil {
					return fmt.Errorf("publish share rename: %w", err)
				}
			}
		}

		return provider.SaveJSON(ctx, a.storageApp, shareFilename, a.shares)
	})

	return err
}

func (a *App) deleteItem(ctx context.Context, item absto.Item) error {
	_, err := a.Exclusive(ctx, item.ID, exclusive.Duration, func(_ context.Context) error {
		for id, share := range a.shares {
			if strings.HasPrefix(share.Path, item.Pathname) {
				if err := a.delete(ctx, id); err != nil {
					return fmt.Errorf("delete share `%s`: %w", id, err)
				}
			}
		}

		return nil
	})

	return err
}
