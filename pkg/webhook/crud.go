package webhook

import (
	"context"
	"fmt"
	"sort"

	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/httputils/v4/pkg/sha"
	"github.com/ViBiOh/httputils/v4/pkg/uuid"
)

func (a *App) generateID() (string, error) {
	for {
		id, err := uuid.New()
		if err != nil {
			return "", err
		}
		idSha := sha.New(id)[:8]

		if _, ok := a.webhooks[idSha]; !ok {
			return idSha, nil
		}
	}
}

func (a *App) List() (output []provider.Webhook) {
	a.mutex.RLock()
	defer a.mutex.RUnlock()

	output = make([]provider.Webhook, 0, len(a.webhooks))

	for _, value := range a.webhooks {
		index := sort.Search(len(output), func(i int) bool {
			return output[i].ID > value.ID
		})

		output = append(output, value)
		copy(output[index+1:], output[index:])
		output[index] = value
	}

	return output
}

func (a *App) Create(ctx context.Context, pathname string, recursive bool, kind provider.WebhookKind, url string, types []provider.EventType) (string, error) {
	var id string

	return id, a.Exclusive(ctx, "create", func(ctx context.Context) (err error) {
		id, err = a.generateID()
		if err != nil {
			return fmt.Errorf("generate id: %w", err)
		}

		webhook := provider.Webhook{
			ID:        id,
			Pathname:  pathname,
			Recursive: recursive,
			Kind:      kind,
			URL:       url,
			Types:     types,
		}

		a.webhooks[id] = webhook

		if err = provider.SaveJSON(ctx, a.storageApp, webhookFilename, a.webhooks); err != nil {
			return fmt.Errorf("save webhooks: %w", err)
		}

		if err = a.redisClient.PublishJSON(ctx, a.pubsubChannel, webhook); err != nil {
			return fmt.Errorf("publish webhook creation: %w", err)
		}

		return nil
	})
}

func (a *App) Delete(ctx context.Context, id string) error {
	return a.Exclusive(ctx, id, func(_ context.Context) error {
		return a.delete(ctx, id)
	})
}

func (a *App) delete(ctx context.Context, id string) error {
	delete(a.webhooks, id)

	if err := provider.SaveJSON(ctx, a.storageApp, webhookFilename, a.webhooks); err != nil {
		return fmt.Errorf("save webhooks: %w", err)
	}

	if err := a.redisClient.PublishJSON(ctx, a.pubsubChannel, provider.Webhook{ID: id}); err != nil {
		return fmt.Errorf("publish webhook deletion: %w", err)
	}

	return nil
}
