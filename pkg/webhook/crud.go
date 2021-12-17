package webhook

import (
	"context"
	"fmt"

	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/httputils/v4/pkg/sha"
	"github.com/ViBiOh/httputils/v4/pkg/uuid"
)

func (a *App) generateID() (string, error) {
	for {
		uuid, err := uuid.New()
		if err != nil {
			return "", err
		}
		id := sha.New(uuid)[:8]

		if _, ok := a.webhooks[id]; !ok {
			return id, nil
		}
	}
}

// List webhooks
func (a *App) List() map[string]provider.Webhook {
	if !a.Enabled() {
		return nil
	}

	a.RLock()
	defer a.RUnlock()

	return a.webhooks
}

// Create a webhook
func (a *App) Create(pathname string, recursive bool, kind provider.WebhookKind, url string, types []provider.EventType) (string, error) {
	if !a.Enabled() {
		return "", fmt.Errorf("webhook is disabled")
	}

	var id string

	return id, a.Exclusive(context.Background(), a.amqpExclusiveRoutingKey, semaphoreDuration, func(_ context.Context) (err error) {
		id, err = a.generateID()
		if err != nil {
			return fmt.Errorf("unable to generate id: %s", err)
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

		if err = provider.SaveJSON(a.storageApp, webhookFilename, a.webhooks); err != nil {
			return fmt.Errorf("unable to save webhooks: %s", err)
		}

		if a.amqpClient != nil {
			if err = a.amqpClient.PublishJSON(webhook, a.amqpExchange, a.amqpRoutingKey); err != nil {
				return fmt.Errorf("unable to publish webhook creation: %s", err)
			}
		}

		return nil
	})
}

// Delete a webhook
func (a *App) Delete(id string) error {
	if !a.Enabled() {
		return fmt.Errorf("webhook is disabled")
	}

	return a.Exclusive(context.Background(), a.amqpExclusiveRoutingKey, semaphoreDuration, func(_ context.Context) error {
		delete(a.webhooks, id)

		if err := provider.SaveJSON(a.storageApp, webhookFilename, a.webhooks); err != nil {
			return fmt.Errorf("unable to save webhooks: %s", err)
		}

		if a.amqpClient != nil {
			if err := a.amqpClient.PublishJSON(provider.Webhook{ID: id}, a.amqpExchange, a.amqpRoutingKey); err != nil {
				return fmt.Errorf("unable to publish webhook deletion: %s", err)
			}
		}

		return nil
	})
}
