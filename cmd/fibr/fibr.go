package main

import (
	"context"
	"embed"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"

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

func newLoginApp(tracerProvider trace.TracerProvider, basicConfig basicMemory.Config) provider.Auth {
	basicApp, err := basicMemory.New(basicConfig)
	if err != nil {
		slog.Error("auth memory", "err", err)
		os.Exit(1)
	}

	basicProviderProvider := basic.New(basicApp, "fibr")
	return authMiddleware.New(basicApp, tracerProvider, basicProviderProvider)
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

	stopOnDone := Starters{services.amqpThumbnailApp, services.amqpExifApp, services.sanitizerApp}
	stopOnDone.Start(clients.health.Done(ctx))
	defer stopOnDone.GracefulWait()

	stopOnEnd := Starters{services.webhookApp, services.shareApp}
	stopOnEnd.Start(endCtx)
	defer stopOnEnd.GracefulWait()

	ports.Start(endCtx)
	defer ports.GracefulWait()

	go adapters.eventBus.Start(endCtx, adapters.storageApp, []provider.Renamer{services.thumbnailApp.Rename, services.metadataApp.Rename}, services.shareApp.EventConsumer, services.thumbnailApp.EventConsumer, services.metadataApp.EventConsumer, services.webhookApp.EventConsumer)

	clients.health.WaitForTermination(ports.TerminateOnDone())
}
