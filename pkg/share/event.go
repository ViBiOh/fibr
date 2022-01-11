package share

import (
	"context"
	"fmt"
	"strings"

	absto "github.com/ViBiOh/absto/pkg/model"
	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/httputils/v4/pkg/logger"
)

// EventConsumer handle event pushed to the event bus
func (a *App) EventConsumer(e provider.Event) {
	switch e.Type {
	case provider.RenameEvent:
		if err := a.renameItem(e.Item, *e.New); err != nil {
			logger.Error("unable to rename share: %s", err)
		}
	case provider.DeleteEvent:
		if err := a.deleteItem(e.Item); err != nil {
			logger.Error("unable to rename share: %s", err)
		}
	}
}

func (a *App) renameItem(old, new absto.Item) error {
	_, err := a.Exclusive(context.Background(), a.amqpExclusiveRoutingKey, semaphoreDuration, func(_ context.Context) error {
		for id, share := range a.shares {
			if strings.HasPrefix(share.Path, old.Pathname) {
				share.Path = strings.Replace(share.Path, old.Pathname, new.Pathname, 1)
				a.shares[id] = share

				if a.amqpClient != nil {
					if err := a.amqpClient.PublishJSON(share, a.amqpExchange, a.amqpRoutingKey); err != nil {
						return fmt.Errorf("unable to publish share rename: %s", err)
					}
				}
			}
		}

		return provider.SaveJSON(a.storageApp, shareFilename, a.shares)
	})

	return err
}

func (a *App) deleteItem(item absto.Item) error {
	_, err := a.Exclusive(context.Background(), a.amqpExclusiveRoutingKey, semaphoreDuration, func(_ context.Context) error {
		for id, share := range a.shares {
			if strings.HasPrefix(share.Path, item.Pathname) {
				if err := a.delete(id); err != nil {
					return fmt.Errorf("unable to delete share `%s`: %s", id, err)
				}
			}
		}

		return nil
	})

	return err
}
