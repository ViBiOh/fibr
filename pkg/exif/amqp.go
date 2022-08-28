package exif

import (
	"context"
	"encoding/json"
	"fmt"

	absto "github.com/ViBiOh/absto/pkg/model"
	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/httputils/v4/pkg/logger"
	"github.com/ViBiOh/httputils/v4/pkg/tracer"
	"github.com/streadway/amqp"
)

func (a App) AMQPHandler(ctx context.Context, message amqp.Delivery) error {
	ctx, end := tracer.StartSpan(ctx, a.tracer, "amqp")
	defer end()

	var resp provider.ExifResponse

	if err := json.Unmarshal(message.Body, &resp); err != nil {
		return fmt.Errorf("decode: %w", err)
	}

	if resp.Exif.IsZero() {
		logger.WithField("item", resp.Item.Pathname).Debug("no exif")
		return nil
	}

	exif, err := a.GetExifFor(ctx, resp.Item)
	if err != nil && !absto.IsNotExist(err) {
		logger.WithField("item", resp.Item.Pathname).Error("load exif: %s", err)
	}

	resp.Exif.Description = exif.Description

	if err := a.SaveExifFor(ctx, resp.Item, resp.Exif); err != nil {
		return fmt.Errorf("save: %w", err)
	}

	return a.processExif(ctx, resp.Item, resp.Exif, true)
}
