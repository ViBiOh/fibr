package exif

import (
	"context"
	"encoding/json"
	"fmt"

	absto "github.com/ViBiOh/absto/pkg/model"
	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/httputils/v4/pkg/logger"
	"github.com/streadway/amqp"
)

// AmqpHandler handle exif message
func (a App) AmqpHandler(message amqp.Delivery) error {
	var resp provider.ExifResponse

	if err := json.Unmarshal(message.Body, &resp); err != nil {
		return fmt.Errorf("decode: %s", err)
	}

	if resp.Exif.IsZero() {
		logger.Debug("[exif] [handler] %s: no exif", resp.Item.Pathname)
		return nil
	}

	ctx := context.Background()

	exif, err := a.loadExif(ctx, resp.Item)
	if err != nil && !absto.IsNotExist(err) {
		logger.WithField("item", resp.Item.Pathname).Error("load exif: %s", err)
	}

	resp.Exif.Description = exif.Description

	if err := a.saveMetadata(ctx, resp.Item, resp.Exif); err != nil {
		return fmt.Errorf("save: %s", err)
	}

	return a.processExif(ctx, resp.Item, resp.Exif, true)
}
