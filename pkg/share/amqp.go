package share

import (
	"encoding/json"
	"fmt"

	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/streadway/amqp"
)

func (a *App) AMQPHandler(message amqp.Delivery) error {
	var share provider.Share

	if err := json.Unmarshal(message.Body, &share); err != nil {
		return fmt.Errorf("decode: %s", err)
	}

	a.Lock()
	defer a.Unlock()

	if share.Creation.IsZero() {
		delete(a.shares, share.ID)
	} else {
		a.shares[share.ID] = share
	}

	return nil
}
