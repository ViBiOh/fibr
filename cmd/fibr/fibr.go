package main

import (
	"context"

	"github.com/ViBiOh/httputils/v4/pkg/alcotest"
	"github.com/ViBiOh/httputils/v4/pkg/httputils"
	"github.com/ViBiOh/httputils/v4/pkg/logger"
	"github.com/ViBiOh/httputils/v4/pkg/owasp"
	"github.com/ViBiOh/httputils/v4/pkg/server"
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

	endCtx := clients.health.EndCtx()

	services, err := newServices(endCtx, config, clients, adapters)
	logger.FatalfOnErr(ctx, err, "services")

	go services.Start(adapters, clients.health.DoneCtx(), endCtx)
	defer services.Close()

	port := newPort(config, services)

	go services.server.Start(endCtx, httputils.Handler(port, clients.health, clients.telemetry.Middleware("http"), owasp.New(config.owasp).Middleware))

	clients.health.WaitForTermination(services.server.Done())

	server.GracefulWait(services.server.Done())
}
