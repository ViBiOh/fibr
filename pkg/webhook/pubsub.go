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

	s.webhooks.Patch(func(webhooks map[string]provider.Webhook) {
		if len(webhook.URL) == 0 {
			delete(webhooks, webhook.ID)
		} else {
			webhooks[webhook.ID] = webhook
		}
	})
}
