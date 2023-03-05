package metadata

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/httputils/v4/pkg/logger"
	"github.com/ViBiOh/httputils/v4/pkg/tracer"
	"github.com/streadway/amqp"
)

func (a App) AMQPHandler(ctx context.Context, message amqp.Delivery) error {
	var err error

	ctx, end := tracer.StartSpan(ctx, a.tracer, "amqp")
	defer end(&err)

	var resp provider.ExifResponse

	if err = json.Unmarshal(message.Body, &resp); err != nil {
		return fmt.Errorf("decode: %w", err)
	}

	if resp.Exif.IsZero() {
		logger.WithField("item", resp.Item.Pathname).Debug("no exif")
		return nil
	}

	metadata, err := a.Update(ctx, resp.Item, provider.ReplaceExif(resp.Exif))
	if err != nil {
		return fmt.Errorf("update: %w", err)
	}

	return a.processMetadata(ctx, resp.Item, metadata, true)
}
