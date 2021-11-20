package share

import (
	"context"
	"fmt"
	"strings"

	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/httputils/v4/pkg/logger"
)

// EventConsumer handle event pushed to the event bus
func (a *App) EventConsumer(e provider.Event) {
	if !a.Enabled() {
		return
	}

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

func (a *App) renameItem(old, new provider.StorageItem) error {
	return a.Exclusive(context.Background(), a.amqpExclusiveRoutingKey, semaphoreDuration, func(_ context.Context) error {
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
}

func (a *App) deleteItem(item provider.StorageItem) error {
	return a.Exclusive(context.Background(), a.amqpExclusiveRoutingKey, semaphoreDuration, func(_ context.Context) error {
		for id, share := range a.shares {
			if strings.HasPrefix(share.Path, item.Pathname) {
				delete(a.shares, id)

				if a.amqpClient != nil {
					if err := a.amqpClient.PublishJSON(provider.Share{ID: id}, a.amqpExchange, a.amqpRoutingKey); err != nil {
						return fmt.Errorf("unable to publish share deletion: %s", err)
					}
				}
			}
		}

		return provider.SaveJSON(a.storageApp, shareFilename, a.shares)
	})
}
