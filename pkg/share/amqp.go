package share

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/streadway/amqp"
)

func (a *App) AMQPHandler(_ context.Context, message amqp.Delivery) error {
	var share provider.Share

	if err := json.Unmarshal(message.Body, &share); err != nil {
		return fmt.Errorf("decode: %w", err)
	}

	a.mutex.Lock()
	defer a.mutex.Unlock()

	if share.Creation.IsZero() {
		delete(a.shares, share.ID)
	} else {
		a.shares[share.ID] = share
	}

	return nil
}
