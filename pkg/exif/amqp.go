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

	if err := a.saveMetadata(resp.Item, exifMetadataFilename, resp.Exif); err != nil {
		return fmt.Errorf("unable to save exif: %s", err)
	}

	return a.processExif(resp.Item, resp.Exif)
}