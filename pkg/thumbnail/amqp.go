package thumbnail

import (
	"context"
	"encoding/json"
	"fmt"

	absto "github.com/ViBiOh/absto/pkg/model"
	"github.com/ViBiOh/httputils/v4/pkg/telemetry"
	vith "github.com/ViBiOh/vith/pkg/model"
	amqp "github.com/rabbitmq/amqp091-go"
)

func (s Service) AMQPHandler(ctx context.Context, message amqp.Delivery) (err error) {
	ctx, end := telemetry.StartSpan(ctx, s.tracer, "amqp")
	defer end(&err)

	var req vith.Request
	if err := json.Unmarshal(message.Body, &req); err != nil {
		return fmt.Errorf("decode: %w", err)
	}

	key := redisKey(s.PathForScale(absto.Item{
		ID:       absto.ID(req.Input),
		Pathname: req.Input,
	}, req.Scale))

	return s.redisClient.Delete(ctx, key)
}
