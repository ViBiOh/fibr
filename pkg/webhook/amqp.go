package webhook

import (
	"encoding/json"
	"fmt"

	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/streadway/amqp"
)

// AMQPHandler handle exif message
func (a *App) AMQPHandler(message amqp.Delivery) error {
	var webhook provider.Webhook

	if err := json.Unmarshal(message.Body, &webhook); err != nil {
		return fmt.Errorf("decode: %w", err)
	}

	a.Lock()
	defer a.Unlock()

	if len(webhook.URL) == 0 {
		delete(a.webhooks, webhook.ID)
	} else {
		a.webhooks[webhook.ID] = webhook
	}

	return nil
}
