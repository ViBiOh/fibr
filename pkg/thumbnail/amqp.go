package thumbnail

import (
	"context"
	"encoding/json"
	"fmt"

	absto "github.com/ViBiOh/absto/pkg/model"
	"github.com/ViBiOh/httputils/v4/pkg/logger"
	"github.com/ViBiOh/httputils/v4/pkg/tracer"
	vith "github.com/ViBiOh/vith/pkg/model"
	"github.com/streadway/amqp"
)

func (a App) AMQPHandler(ctx context.Context, message amqp.Delivery) error {
	ctx, end := tracer.StartSpan(ctx, a.tracer, "amqp")
	defer end()

	var req vith.Request
	if err := json.Unmarshal(message.Body, &req); err != nil {
		return fmt.Errorf("decode: %w", err)
	}

	key := redisKey(a.PathForScale(absto.Item{
		ID:       absto.ID(req.Input),
		Pathname: req.Input,
	}, req.Scale))

	logger.Info("evicting key=`%s` for input `%s` and scale=%d", key, req.Input, req.Scale)

	return a.redisClient.Delete(ctx, key)
}
