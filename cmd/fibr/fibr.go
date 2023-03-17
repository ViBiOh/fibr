package main

import (
	"context"
	"embed"
	"fmt"
	"net/http"

	_ "net/http/pprof"

	"github.com/ViBiOh/absto/pkg/absto"
	"github.com/ViBiOh/auth/v2/pkg/ident/basic"
	authMiddleware "github.com/ViBiOh/auth/v2/pkg/middleware"
	basicMemory "github.com/ViBiOh/auth/v2/pkg/store/memory"
	"github.com/ViBiOh/fibr/pkg/crud"
	"github.com/ViBiOh/fibr/pkg/exclusive"
	"github.com/ViBiOh/fibr/pkg/fibr"
	"github.com/ViBiOh/fibr/pkg/metadata"
	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/fibr/pkg/sanitizer"
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

func main() {
	config, err := newConfig()
	if err != nil {
		logger.Fatal(fmt.Errorf("config: %w", err))
	}

	alcotest.DoAndExit(config.alcotest)

	go func() {
		fmt.Println(http.ListenAndServe("localhost:9999", http.DefaultServeMux))
	}()

	ctx := context.Background()

	client, err := newClient(ctx, config)
	if err != nil {
		logger.Fatal(fmt.Errorf("client: %w", err))
	}

	defer client.Close(ctx)

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

	var exclusiveApp exclusive.App
	if client.redis.Enabled() {
		exclusiveApp = exclusive.New(client.redis)
	}

	metadataApp, err := metadata.New(config.metadata, storageApp, prometheusRegisterer, client.tracer, client.amqp, client.redis, exclusiveApp)
	logger.Fatal(err)

	webhookApp := webhook.New(config.webhook, storageApp, prometheusRegisterer, client.redis, rendererApp, thumbnailApp, exclusiveApp)

	shareApp, err := share.New(config.share, storageApp, client.redis, exclusiveApp)
	logger.Fatal(err)

	amqpThumbnailApp, err := amqphandler.New(config.amqpThumbnail, client.amqp, client.tracer.GetTracer("amqp_handler_thumbnail"), thumbnailApp.AMQPHandler)
	logger.Fatal(err)

	amqpExifApp, err := amqphandler.New(config.amqpExif, client.amqp, client.tracer.GetTracer("amqp_handler_exif"), metadataApp.AMQPHandler)
	logger.Fatal(err)

	searchApp := search.New(filteredStorage, thumbnailApp, metadataApp, exclusiveApp, client.tracer.GetTracer("search"))

	crudApp, err := crud.New(config.crud, storageApp, filteredStorage, rendererApp, shareApp, webhookApp, thumbnailApp, metadataApp, searchApp, eventBus.Push, client.tracer.GetTracer("crud"))
	logger.Fatal(err)

	sanitizerApp := sanitizer.New(config.sanitizer, filteredStorage, exclusiveApp, crudApp, eventBus.Push)

	var middlewareApp provider.Auth
	if !*config.disableAuth {
		middlewareApp = newLoginApp(client.tracer.GetTracer("auth"), config.basic)
	}

	fibrApp := fibr.New(&crudApp, rendererApp, shareApp, webhookApp, middlewareApp)
	handler := rendererApp.Handler(fibrApp.TemplateFunc)

	doneCtx := client.health.Done(ctx)
	endCtx := client.health.End(ctx)

	go amqpThumbnailApp.Start(doneCtx)
	go amqpExifApp.Start(doneCtx)
	go webhookApp.Start(endCtx)
	go shareApp.Start(endCtx)
	go sanitizerApp.Start(endCtx)
	go eventBus.Start(endCtx, storageApp, []provider.Renamer{thumbnailApp.Rename, metadataApp.Rename}, shareApp.EventConsumer, thumbnailApp.EventConsumer, metadataApp.EventConsumer, webhookApp.EventConsumer)

	go promServer.Start(endCtx, "prometheus", client.prometheus.Handler())
	go appServer.Start(endCtx, "http", httputils.Handler(handler, client.health, recoverer.Middleware, client.prometheus.Middleware, client.tracer.Middleware, owasp.New(config.owasp).Middleware))

	client.health.WaitForTermination(appServer.Done())
	server.GracefulWait(appServer.Done(), promServer.Done(), amqpExifApp.Done(), eventBus.Done(), webhookApp.Done(), shareApp.Done())
}
