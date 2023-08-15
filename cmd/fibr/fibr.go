package main

import (
	"context"
	"embed"
	"fmt"
	"log"
	"net/http"

	_ "net/http/pprof"

	"github.com/ViBiOh/auth/v2/pkg/ident/basic"
	authMiddleware "github.com/ViBiOh/auth/v2/pkg/middleware"
	basicMemory "github.com/ViBiOh/auth/v2/pkg/store/memory"
	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/httputils/v4/pkg/alcotest"
	"github.com/ViBiOh/httputils/v4/pkg/logger"
	"go.opentelemetry.io/otel/trace"
)

//go:embed templates static
var content embed.FS

func newLoginApp(tracer trace.Tracer, basicConfig basicMemory.Config) provider.Auth {
	basicApp, err := basicMemory.New(basicConfig)
	logger.Fatal(err)

	basicProviderProvider := basic.New(basicApp, "fibr")
	return authMiddleware.New(basicApp, tracer, basicProviderProvider)
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
		logger.Fatal(fmt.Errorf("clients: %w", err))
	}

	defer clients.Close(ctx)

	adapters, err := newAdapters(config, clients)
	if err != nil {
		logger.Fatal(fmt.Errorf("adapters: %w", err))
	}

	services, err := newServices(config, clients, adapters)
	if err != nil {
		logger.Fatal(fmt.Errorf("services: %w", err))
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
