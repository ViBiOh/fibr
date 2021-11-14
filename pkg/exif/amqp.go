package exif

import (
	"encoding/json"
	"fmt"

	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/streadway/amqp"
)

// AmqpHandler handle exif message
func (a App) AmqpHandler(message amqp.Delivery) error {
	var resp provider.ExifResponse

	if err := json.Unmarshal(message.Body, &resp); err != nil {
		return fmt.Errorf("unable to decode exif: %s", err)
	}

	if err := a.updateDate(resp.Item, resp.Exif); err != nil {
		return fmt.Errorf("unable to update date: %s", err)
	}

	if err := a.aggregate(resp.Item); err != nil {
		return fmt.Errorf("unable to aggregate exif: %s", err)
	}

	return nil
}
