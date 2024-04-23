package main

import (
	"context"
	"embed"
	"fmt"
	"log/slog"
	"net/http"
	"os"

	_ "net/http/pprof"

	"github.com/ViBiOh/auth/v2/pkg/ident/basic"
	authMiddleware "github.com/ViBiOh/auth/v2/pkg/middleware"
	basicMemory "github.com/ViBiOh/auth/v2/pkg/store/memory"
	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/httputils/v4/pkg/alcotest"
	"github.com/ViBiOh/httputils/v4/pkg/httputils"
	"github.com/ViBiOh/httputils/v4/pkg/logger"
	"github.com/ViBiOh/httputils/v4/pkg/owasp"
	"github.com/ViBiOh/httputils/v4/pkg/recoverer"
	"github.com/ViBiOh/httputils/v4/pkg/server"
	"go.opentelemetry.io/otel/trace"
)

//go:embed templates static
var content embed.FS

func newLoginService(tracerProvider trace.TracerProvider, basicConfig *basicMemory.Config) provider.Auth {
	basicService, err := basicMemory.New(basicConfig)
	if err != nil {
		slog.LogAttrs(context.Background(), slog.LevelError, "auth memory", slog.Any("error", err))
		os.Exit(1)
	}

	basicProviderProvider := basic.New(basicService, "fibr")
	return authMiddleware.New(basicService, tracerProvider, basicProviderProvider)
}

func main() {
	config := newConfig()
	alcotest.DoAndExit(config.alcotest)

	go func() {
		fmt.Println(http.ListenAndServe("localhost:9999", http.DefaultServeMux))
	}()

	ctx := context.Background()

	clients, err := newClient(ctx, config)
	logger.FatalfOnErr(ctx, err, "clients")

	defer clients.Close(ctx)

	adapters, err := newAdapters(config, clients)
	logger.FatalfOnErr(ctx, err, "adapters")

	endCtx := clients.health.EndCtx()

	services, err := newServices(endCtx, config, clients, adapters)
	logger.FatalfOnErr(ctx, err, "services")

	stopOnDone := Starters{services.amqpThumbnail, services.amqpExif, services.sanitizer}
	stopOnDone.Start(clients.health.DoneCtx())
	defer stopOnDone.GracefulWait()

	stopOnEnd := Starters{services.webhook, services.share}
	stopOnEnd.Start(endCtx)
	defer stopOnEnd.GracefulWait()

	go adapters.eventBus.Start(endCtx, adapters.storage, []provider.Renamer{services.thumbnail.Rename, services.metadata.Rename}, services.share.EventConsumer, services.thumbnail.EventConsumer, services.metadata.EventConsumer, services.webhook.EventConsumer)

	appServer := server.New(config.appServer)

	go appServer.Start(endCtx, httputils.Handler(services.renderer.Handler(services.fibr.TemplateFunc), clients.health, recoverer.Middleware, clients.telemetry.Middleware("http"), owasp.New(config.owasp).Middleware))

	clients.health.WaitForTermination(appServer.Done())

	server.GracefulWait(appServer.Done())
}
