package main

import (
	"context"
	"errors"
	"fmt"

	"github.com/ViBiOh/httputils/v4/pkg/amqp"
	"github.com/ViBiOh/httputils/v4/pkg/health"
	"github.com/ViBiOh/httputils/v4/pkg/logger"
	"github.com/ViBiOh/httputils/v4/pkg/pprof"
	"github.com/ViBiOh/httputils/v4/pkg/redis"
	"github.com/ViBiOh/httputils/v4/pkg/request"
	"github.com/ViBiOh/httputils/v4/pkg/telemetry"
)

type client struct {
	redis     redis.Client
	amqp      *amqp.Client
	health    *health.Service
	telemetry telemetry.Service
	pprof     pprof.Service
}

func newClient(ctx context.Context, config configuration) (client, error) {
	var (
		output client
		err    error
	)

	logger.Init(ctx, config.logger)

	output.telemetry, err = telemetry.New(ctx, config.telemetry)
	if err != nil {
		return output, fmt.Errorf("telemetry: %w", err)
	}

	service, version, env := output.telemetry.GetServiceVersionAndEnv()
	output.pprof = pprof.New(config.pprof, service, version, env)

	logger.AddOpenTelemetryToDefaultLogger(output.telemetry)
	request.AddOpenTelemetryToDefaultClient(output.telemetry.MeterProvider(), output.telemetry.TracerProvider())

	output.health = health.New(ctx, config.health)

	output.redis, err = redis.New(ctx, config.redis, output.telemetry.MeterProvider(), output.telemetry.TracerProvider())
	if err != nil {
		return output, fmt.Errorf("redis: %w", err)
	}

	output.amqp, err = amqp.New(ctx, config.amqp, output.telemetry.MeterProvider(), output.telemetry.TracerProvider())
	if err != nil && !errors.Is(err, amqp.ErrNoConfig) {
		return output, fmt.Errorf("amqp: %w", err)
	}

	return output, nil
}

func (c client) Start() {
	go c.pprof.Start(c.health.DoneCtx())
}

func (c client) Close(ctx context.Context) {
	c.amqp.Close(ctx)
	c.redis.Close(ctx)
	c.telemetry.Close(ctx)
}
