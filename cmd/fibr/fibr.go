package main

import (
	"context"
	"embed"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"syscall"

	_ "net/http/pprof"

	"github.com/ViBiOh/auth/v2/pkg/ident/basic"
	authMiddleware "github.com/ViBiOh/auth/v2/pkg/middleware"
	basicMemory "github.com/ViBiOh/auth/v2/pkg/store/memory"
	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/httputils/v4/pkg/alcotest"
	"go.opentelemetry.io/otel/trace"
)

//go:embed templates static
var content embed.FS

func newLoginService(tracerProvider trace.TracerProvider, basicConfig *basicMemory.Config) provider.Auth {
	basicService, err := basicMemory.New(basicConfig)
	if err != nil {
		slog.Error("auth memory", "err", err)
		os.Exit(1)
	}

	basicProviderProvider := basic.New(basicService, "fibr")
	return authMiddleware.New(basicService, tracerProvider, basicProviderProvider)
}

func main() {
	config, err := newConfig()
	if err != nil {
		log.Fatal(fmt.Errorf("config: %s", err))
	}

	alcotest.DoAndExit(config.alcotest)

	go func() {
		fmt.Println(http.ListenAndServe("localhost:9999", http.DefaultServeMux))
	}()

	ctx := context.Background()

	clients, err := newClient(ctx, config)
	if err != nil {
		slog.Error("clients", "err", err)
		os.Exit(1)
	}

	defer clients.Close(ctx)

	adapters, err := newAdapters(config, clients)
	if err != nil {
		slog.Error("adapters", "err", err)
		os.Exit(1)
	}

	services, err := newServices(config, clients, adapters)
	if err != nil {
		slog.Error("services", "err", err)
		os.Exit(1)
	}

	ports := newPorts(config, clients, services)

	endCtx := clients.health.End(ctx)

	stopOnDone := Starters{services.amqpThumbnail, services.amqpExif, services.sanitizer}
	stopOnDone.Start(clients.health.Done(ctx))
	defer stopOnDone.GracefulWait()

	stopOnEnd := Starters{services.webhook, services.share}
	stopOnEnd.Start(endCtx)
	defer stopOnEnd.GracefulWait()

	ports.Start(endCtx)
	defer ports.GracefulWait()

	go adapters.eventBus.Start(endCtx, adapters.storage, []provider.Renamer{services.thumbnail.Rename, services.metadata.Rename}, services.share.EventConsumer, services.thumbnail.EventConsumer, services.metadata.EventConsumer, services.webhook.EventConsumer)

	clients.health.WaitForTermination(ports.TerminateOnDone(), syscall.SIGTERM)
}
