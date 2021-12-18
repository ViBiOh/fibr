package share

import (
	"encoding/json"
	"fmt"

	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/streadway/amqp"
)

// AmqpHandler handle exif message
func (a *App) AmqpHandler(message amqp.Delivery) error {
	var share provider.Share

	if err := json.Unmarshal(message.Body, &share); err != nil {
		return fmt.Errorf("unable to decode: %s", err)
	}

	a.Lock()

	if share.Creation.IsZero() {
		delete(a.shares, share.ID)
	} else {
		a.shares[share.ID] = share
	}

	a.Unlock()

	return nil
}
