package webhook

import (
	"log/slog"

	"github.com/ViBiOh/fibr/pkg/provider"
)

func (s *Service) PubSubHandle(webhook provider.Webhook, err error) {
	if err != nil {
		slog.Error("Webhook's PubSub", "error", err)
		return
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	slog.Info("Webhook's PubSub", "id", webhook.ID)

	if len(webhook.URL) == 0 {
		delete(s.webhooks, webhook.ID)
	} else {
		s.webhooks[webhook.ID] = webhook
	}
}
