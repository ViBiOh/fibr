package main

import (
	"context"

	"github.com/ViBiOh/httputils/v4/pkg/alcotest"
	"github.com/ViBiOh/httputils/v4/pkg/health"
	"github.com/ViBiOh/httputils/v4/pkg/logger"
)

func main() {
	config := newConfig()
	alcotest.DoAndExit(config.alcotest)

	ctx := context.Background()

	clients, err := newClients(ctx, config)
	logger.FatalfOnErr(ctx, err, "clients")

	go clients.Start()
	defer clients.Close(ctx)

	adapters, err := newAdapters(config, clients)
	logger.FatalfOnErr(ctx, err, "adapters")

	services, err := newServices(clients.health.EndCtx(), config, clients, adapters)
	logger.FatalfOnErr(ctx, err, "services")

	go services.Start(adapters, clients.health.DoneCtx(), clients.health.EndCtx())
	defer services.Close()

	port := newPort(clients, services)

	go services.server.Start(clients.health.EndCtx(), port)

	clients.health.WaitForTermination(services.server.Done())
	health.WaitAll(services.server.Done())
}
