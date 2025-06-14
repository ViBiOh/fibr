package webhook

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/ViBiOh/fibr/pkg/provider"
)

func (s *Service) generateID() (string, error) {
	for {
		idSha := provider.Hash(provider.Identifier())[:8]

		if _, ok := s.webhooks[idSha]; !ok {
			return idSha, nil
		}
	}
}

func (s *Service) List() (output []provider.Webhook) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	output = make([]provider.Webhook, 0, len(s.webhooks))

	for _, webhook := range s.webhooks {
		index := sort.Search(len(output), func(i int) bool {
			return output[i].ID > webhook.ID
		})

		output = append(output, webhook)
		copy(output[index+1:], output[index:])
		output[index] = webhook
	}

	return output
}

func (s *Service) Find(url string, request provider.Request) (output []provider.Webhook) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	output = make([]provider.Webhook, 0, len(s.webhooks))

	for _, webhook := range s.webhooks {
		if webhook.URL == url && webhook.Pathname == request.Filepath() {
			output = append(output, webhook)
		}
	}

	return output
}

func (s *Service) Create(ctx context.Context, pathname string, recursive bool, kind provider.WebhookKind, url string, types []provider.EventType) (string, error) {
	var id string

	return id, s.Exclusive(ctx, "create", func(ctx context.Context) (err error) {
		id, err = s.generateID()
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
			Created:   time.Now(),
		}

		for existingID, existing := range s.webhooks {
			if webhook.Similar(existing) {
				id = existingID
				return nil
			}
		}

		s.webhooks[id] = webhook

		if err = provider.SaveJSON(ctx, s.storage, webhookFilename, s.webhooks); err != nil {
			return fmt.Errorf("save webhooks: %w", err)
		}

		if err = s.redisClient.PublishJSON(ctx, s.pubsubChannel, webhook); err != nil {
			return fmt.Errorf("publish webhook creation: %w", err)
		}

		return nil
	})
}

func (s *Service) Delete(ctx context.Context, id string) error {
	return s.Exclusive(ctx, id, func(_ context.Context) error {
		return s.delete(ctx, id)
	})
}

func (s *Service) delete(ctx context.Context, id string) error {
	delete(s.webhooks, id)

	if err := provider.SaveJSON(ctx, s.storage, webhookFilename, s.webhooks); err != nil {
		return fmt.Errorf("save webhooks: %w", err)
	}

	if err := s.redisClient.PublishJSON(ctx, s.pubsubChannel, provider.Webhook{ID: id}); err != nil {
		return fmt.Errorf("publish webhook deletion: %w", err)
	}

	return nil
}
