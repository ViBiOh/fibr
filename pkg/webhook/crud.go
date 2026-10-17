package webhook

import (
	"context"
	"sort"
	"time"

	"github.com/ViBiOh/fibr/pkg/provider"
)

func generateID(webhooks map[string]provider.Webhook) string {
	for {
		idSha := provider.Hash(provider.Identifier())[:8]

		if _, ok := webhooks[idSha]; !ok {
			return idSha
		}
	}
}

func (s *Service) List() []provider.Webhook {
	webhooks := s.webhooks.Items()

	output := make([]provider.Webhook, 0, len(webhooks))

	for _, webhook := range webhooks {
		index := sort.Search(len(output), func(i int) bool {
			return output[i].ID > webhook.ID
		})

		output = append(output, webhook)
		copy(output[index+1:], output[index:])
		output[index] = webhook
	}

	return output
}

func (s *Service) Get(id string) provider.Webhook {
	for _, webhook := range s.webhooks.Items() {
		if webhook.ID == id {
			return webhook
		}
	}

	return provider.Webhook{}
}

func (s *Service) FindByURL(url string) []provider.Webhook {
	var webhooks []provider.Webhook

	for _, webhook := range s.webhooks.Items() {
		if webhook.URL == url {
			webhooks = append(webhooks, webhook)
		}
	}

	return webhooks
}

func (s *Service) Create(ctx context.Context, pathname string, recursive bool, kind provider.WebhookKind, url string, types []provider.EventType) (string, error) {
	var id string

	err := s.webhooks.Update(ctx, func(webhooks map[string]provider.Webhook) ([]provider.Webhook, error) {
		webhook := provider.Webhook{
			ID:        generateID(webhooks),
			Pathname:  pathname,
			Recursive: recursive,
			Kind:      kind,
			URL:       url,
			Types:     types,
			Created:   time.Now(),
		}

		for existingID, existing := range webhooks {
			if webhook.Similar(existing) {
				id = existingID

				return nil, nil
			}
		}

		id = webhook.ID
		webhooks[id] = webhook

		return []provider.Webhook{webhook}, nil
	})

	return id, err
}

func (s *Service) Delete(ctx context.Context, id string) error {
	return s.webhooks.Update(ctx, func(webhooks map[string]provider.Webhook) ([]provider.Webhook, error) {
		delete(webhooks, id)

		return []provider.Webhook{{ID: id}}, nil
	})
}
