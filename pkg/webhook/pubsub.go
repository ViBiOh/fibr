package webhook

import (
	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/httputils/v4/pkg/logger"
)

func (a *App) PubSubHandle(webhook provider.Webhook, err error) {
	if err != nil {
		logger.Error("Webhook's PubSub: %s", err)
		return
	}

	a.mutex.Lock()
	defer a.mutex.Unlock()

	logger.WithField("id", webhook.ID).Info("Webhook's PubSub")

	if len(webhook.URL) == 0 {
		delete(a.webhooks, webhook.ID)
	} else {
		a.webhooks[webhook.ID] = webhook
	}
}
