package main

import (
	"context"
	"errors"
	"fmt"

	"github.com/ViBiOh/httputils/v4/pkg/amqp"
	"github.com/ViBiOh/httputils/v4/pkg/health"
	"github.com/ViBiOh/httputils/v4/pkg/logger"
	"github.com/ViBiOh/httputils/v4/pkg/prometheus"
	"github.com/ViBiOh/httputils/v4/pkg/redis"
	"github.com/ViBiOh/httputils/v4/pkg/request"
	"github.com/ViBiOh/httputils/v4/pkg/tracer"
)

type client struct {
	redis      redis.Client
	tracer     tracer.App
	amqp       *amqp.Client
	prometheus prometheus.App
	logger     logger.Logger
	health     health.App
}

func newClient(ctx context.Context, config configuration) (client, error) {
	var (
		output client
		err    error
	)

	output.logger = logger.New(config.logger)
	logger.Global(output.logger)

	output.tracer, err = tracer.New(ctx, config.tracer)
	if err != nil {
		return output, fmt.Errorf("tracer: %w", err)
	}

	request.AddTracerToDefaultClient(output.tracer.GetProvider())

	output.prometheus = prometheus.New(config.prometheus)
	output.health = health.New(config.health)

	output.redis, err = redis.New(config.redis, output.tracer.GetProvider())
	if err != nil {
		return output, fmt.Errorf("redis: %w", err)
	}

	output.amqp, err = amqp.New(config.amqp, output.prometheus.Registerer(), output.tracer.GetTracer("amqp"))
	if err != nil && !errors.Is(err, amqp.ErrNoConfig) {
		return output, fmt.Errorf("amqp: %w", err)
	}

	return output, nil
}

func (c client) Close(ctx context.Context) {
	c.amqp.Close()
	c.redis.Close()
	c.tracer.Close(ctx)
	c.logger.Close()
}
