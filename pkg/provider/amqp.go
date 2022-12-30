package provider

import (
	"context"
	"time"

	"github.com/ViBiOh/httputils/v4/pkg/amqp"
)

const SemaphoreDuration = time.Second * 10

func Exclusive(ctx context.Context, amqpClient *amqp.Client, name string, action func(ctx context.Context) error) error {
	if amqpClient == nil {
		return action(ctx)
	}

exclusive:
	acquired, err := amqpClient.Exclusive(ctx, name, SemaphoreDuration, func(ctx context.Context) error {
		return action(ctx)
	})
	if err != nil {
		return err
	}

	if !acquired {
		time.Sleep(time.Second)
		goto exclusive
	}

	return nil
}
