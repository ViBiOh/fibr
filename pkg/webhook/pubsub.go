package webhook

import (
	"context"
	"log/slog"

	"github.com/ViBiOh/fibr/pkg/provider"
)

func (s *Service) PubSubHandle(webhook provider.Webhook, err error) {
	if err != nil {
		slog.LogAttrs(context.Background(), slog.LevelError, "Webhook's PubSub", slog.Any("error", err))
		return
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	if len(webhook.URL) == 0 {
		delete(s.webhooks, webhook.ID)
	} else {
		s.webhooks[webhook.ID] = webhook
	}
}
