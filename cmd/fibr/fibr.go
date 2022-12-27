package main

import (
	"context"
	"crypto/rand"
	"embed"
	"fmt"
	"net/http"

	_ "net/http/pprof"

	"github.com/ViBiOh/absto/pkg/absto"
	"github.com/ViBiOh/auth/v2/pkg/ident/basic"
	authMiddleware "github.com/ViBiOh/auth/v2/pkg/middleware"
	basicMemory "github.com/ViBiOh/auth/v2/pkg/store/memory"
	"github.com/ViBiOh/fibr/pkg/crud"
	"github.com/ViBiOh/fibr/pkg/exif"
	"github.com/ViBiOh/fibr/pkg/fibr"
	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/fibr/pkg/search"
	"github.com/ViBiOh/fibr/pkg/share"
	"github.com/ViBiOh/fibr/pkg/storage"
	"github.com/ViBiOh/fibr/pkg/thumbnail"
	"github.com/ViBiOh/fibr/pkg/webhook"
	"github.com/ViBiOh/httputils/v4/pkg/alcotest"
	"github.com/ViBiOh/httputils/v4/pkg/amqphandler"
	"github.com/ViBiOh/httputils/v4/pkg/httputils"
	"github.com/ViBiOh/httputils/v4/pkg/logger"
	"github.com/ViBiOh/httputils/v4/pkg/owasp"
	"github.com/ViBiOh/httputils/v4/pkg/recoverer"
	"github.com/ViBiOh/httputils/v4/pkg/renderer"
	"github.com/ViBiOh/httputils/v4/pkg/server"
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

func generateIdentityName() string {
	raw := make([]byte, 4)
	if _, err := rand.Read(raw); err != nil {
		logger.Error("generate identity name: %s", err)
		return "error"
	}

	return fmt.Sprintf("%x", raw)
}

func main() {
	config, err := newConfig()
	if err != nil {
		logger.Fatal(fmt.Errorf("config: %w", err))
	}

	alcotest.DoAndExit(config.alcotest)

	go func() {
		fmt.Println(http.ListenAndServe("localhost:9999", http.DefaultServeMux))
	}()

	client, err := newClient(config)
	if err != nil {
		logger.Fatal(fmt.Errorf("client: %w", err))
	}

	defer client.Close()

	appServer := server.New(config.appServer)
	promServer := server.New(config.promServer)

	prometheusRegisterer := client.prometheus.Registerer()

	storageApp, err := absto.New(config.absto, client.tracer.GetTracer("storage"))
	logger.Fatal(err)

	filteredStorage, err := storage.Get(config.storage, storageApp)
	logger.Fatal(err)

	eventBus, err := provider.NewEventBus(provider.MaxConcurrency, prometheusRegisterer, client.tracer.GetTracer("bus"))
	logger.Fatal(err)

	thumbnailApp, err := thumbnail.New(config.thumbnail, storageApp, client.redis, prometheusRegisterer, client.tracer, client.amqp)
	logger.Fatal(err)

	rendererApp, err := renderer.New(config.renderer, content, fibr.FuncMap, client.tracer.GetTracer("renderer"))
	logger.Fatal(err)

	exifApp, err := exif.New(config.exif, storageApp, prometheusRegisterer, client.tracer, client.amqp, client.redis)
	logger.Fatal(err)

	webhookApp, err := webhook.New(config.webhook, storageApp, prometheusRegisterer, client.amqp, rendererApp, thumbnailApp)
	logger.Fatal(err)

	shareApp, err := share.New(config.share, storageApp, client.amqp)
	logger.Fatal(err)

	amqpThumbnailApp, err := amqphandler.New(config.amqpThumbnail, client.amqp, client.tracer.GetTracer("amqp_handler_thumbnail"), thumbnailApp.AMQPHandler)
	logger.Fatal(err)

	amqpExifApp, err := amqphandler.New(config.amqpExif, client.amqp, client.tracer.GetTracer("amqp_handler_exif"), exifApp.AMQPHandler)
	logger.Fatal(err)

	amqpShareApp, err := amqphandler.New(config.amqpShare, client.amqp, client.tracer.GetTracer("amqp_handler_share"), shareApp.AMQPHandler)
	logger.Fatal(err)

	amqpWebhookApp, err := amqphandler.New(config.amqpWebhook, client.amqp, client.tracer.GetTracer("amqp_handler_webhook"), webhookApp.AMQPHandler)
	logger.Fatal(err)

	searchApp := search.New(filteredStorage, thumbnailApp, exifApp, client.tracer.GetTracer("search"))

	crudApp, err := crud.New(config.crud, storageApp, filteredStorage, rendererApp, shareApp, webhookApp, thumbnailApp, exifApp, searchApp, eventBus.Push, client.amqp, client.tracer.GetTracer("crud"))
	logger.Fatal(err)

	var middlewareApp provider.Auth
	if !*config.disableAuth {
		middlewareApp = newLoginApp(client.tracer.GetTracer("auth"), config.basic)
	}

	fibrApp := fibr.New(&crudApp, rendererApp, shareApp, webhookApp, middlewareApp)
	handler := rendererApp.Handler(fibrApp.TemplateFunc)

	amqpCtx := context.Background()
	ctx := client.health.Context()

	go amqpThumbnailApp.Start(amqpCtx, client.health.Done())
	go amqpExifApp.Start(amqpCtx, client.health.Done())
	go amqpShareApp.Start(amqpCtx, client.health.Done())
	go amqpWebhookApp.Start(amqpCtx, client.health.Done())
	go webhookApp.Start(ctx)
	go shareApp.Start(ctx)
	go crudApp.Start(ctx)
	go eventBus.Start(ctx, storageApp, []provider.Renamer{thumbnailApp.Rename, exifApp.Rename}, shareApp.EventConsumer, thumbnailApp.EventConsumer, exifApp.EventConsumer, webhookApp.EventConsumer)

	go promServer.Start("prometheus", client.health.End(), client.prometheus.Handler())
	go appServer.Start("http", client.health.End(), httputils.Handler(handler, client.health, recoverer.Middleware, client.prometheus.Middleware, client.tracer.Middleware, owasp.New(config.owasp).Middleware))

	client.health.WaitForTermination(appServer.Done())
	server.GracefulWait(appServer.Done(), promServer.Done(), amqpExifApp.Done(), amqpShareApp.Done(), amqpWebhookApp.Done(), eventBus.Done())
}
