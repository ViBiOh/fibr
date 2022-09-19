package main

import (
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
	redis      redis.App
	tracer     tracer.App
	amqp       *amqp.Client
	prometheus prometheus.App
	logger     logger.Logger
	health     health.App
}

func newClient(config configuration) (client, error) {
	var output client
	var err error

	output.logger = logger.New(config.logger)
	logger.Global(output.logger)

	output.tracer, err = tracer.New(config.tracer)
	if err != nil {
		return output, fmt.Errorf("tracer: %w", err)
	}

	request.AddTracerToDefaultClient(output.tracer.GetProvider())

	output.prometheus = prometheus.New(config.prometheus)
	output.health = health.New(config.health)

	prometheusRegisterer := output.prometheus.Registerer()

	output.redis = redis.New(config.redis, prometheusRegisterer, output.tracer.GetTracer("redis"))

	output.amqp, err = amqp.New(config.amqp, prometheusRegisterer, output.tracer.GetTracer("amqp"))
	if err != nil && !errors.Is(err, amqp.ErrNoConfig) {
		return output, fmt.Errorf("amqp: %w", err)
	}

	return output, nil
}

func (c client) Close() {
	c.amqp.Close()
	c.tracer.Close()
	c.logger.Close()
}
