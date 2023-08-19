package webhook

import (
	"log/slog"

	"github.com/ViBiOh/fibr/pkg/provider"
)

func (a *App) PubSubHandle(webhook provider.Webhook, err error) {
	if err != nil {
		slog.Error("Webhook's PubSub", "err", err)
		return
	}

	a.mutex.Lock()
	defer a.mutex.Unlock()

	slog.Info("Webhook's PubSub", "id", webhook.ID)

	if len(webhook.URL) == 0 {
		delete(a.webhooks, webhook.ID)
	} else {
		a.webhooks[webhook.ID] = webhook
	}
}
