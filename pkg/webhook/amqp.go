package webhook

import (
	"encoding/json"
	"fmt"

	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/streadway/amqp"
)

// AmqpHandler handle exif message
func (a *App) AmqpHandler(message amqp.Delivery) error {
	var webhook provider.Webhook

	if err := json.Unmarshal(message.Body, &webhook); err != nil {
		return fmt.Errorf("unable to decode: %s", err)
	}

	a.Lock()
	defer a.Unlock()

	if len(webhook.Pathname) == 0 {
		delete(a.webhooks, webhook.ID)
	} else {
		a.webhooks[webhook.ID] = webhook
	}

	return nil
}
