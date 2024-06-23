package main

import (
	"context"
	"embed"
	"log/slog"
	"os"

	"github.com/ViBiOh/auth/v2/pkg/ident/basic"
	authMiddleware "github.com/ViBiOh/auth/v2/pkg/middleware"
	basicMemory "github.com/ViBiOh/auth/v2/pkg/store/memory"
	"github.com/ViBiOh/fibr/pkg/crud"
	"github.com/ViBiOh/fibr/pkg/fibr"
	"github.com/ViBiOh/fibr/pkg/metadata"
	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/fibr/pkg/sanitizer"
	"github.com/ViBiOh/fibr/pkg/search"
	"github.com/ViBiOh/fibr/pkg/share"
	"github.com/ViBiOh/fibr/pkg/thumbnail"
	"github.com/ViBiOh/fibr/pkg/webhook"
	"github.com/ViBiOh/httputils/v4/pkg/amqphandler"
	"github.com/ViBiOh/httputils/v4/pkg/renderer"
	"github.com/ViBiOh/httputils/v4/pkg/server"
	"go.opentelemetry.io/otel/trace"
)

//go:embed templates static
var content embed.FS

type services struct {
	fibr          fibr.Service
	eventBus      provider.EventBus
	webhook       *webhook.Service
	share         *share.Service
	renderer      *renderer.Service
	amqpThumbnail *amqphandler.Service
	amqpExif      *amqphandler.Service
	server        *server.Server
	sanitizer     sanitizer.Service
	metadata      metadata.Service
	thumbnail     thumbnail.Service
}

func newServices(ctx context.Context, config configuration, clients clients, adapters adapters) (services, error) {
	eventBus, err := provider.NewEventBus(provider.MaxConcurrency, clients.telemetry.MeterProvider(), clients.telemetry.TracerProvider())
	if err != nil {
		return services{}, err
	}

	thumbnailService, err := thumbnail.New(ctx, config.thumbnail, adapters.storage, clients.redis, clients.telemetry.MeterProvider(), clients.telemetry.TracerProvider(), clients.amqp)
	if err != nil {
		return services{}, err
	}

	rendererService, err := renderer.New(ctx, config.renderer, content, fibr.FuncMap, clients.telemetry.MeterProvider(), clients.telemetry.TracerProvider())
	if err != nil {
		return services{}, err
	}

	metadataService, err := metadata.New(ctx, config.metadata, adapters.storage, clients.telemetry.MeterProvider(), clients.telemetry.TracerProvider(), clients.amqp, clients.redis, adapters.exclusiveService)
	if err != nil {
		return services{}, err
	}

	webhookService := webhook.New(config.webhook, adapters.storage, clients.telemetry.MeterProvider(), clients.redis, rendererService, thumbnailService, adapters.exclusiveService)

	shareService, err := share.New(config.share, adapters.storage, clients.redis, adapters.exclusiveService)
	if err != nil {
		return services{}, err
	}

	amqpThumbnailService, err := amqphandler.New(config.amqpThumbnail, clients.amqp, clients.telemetry.MeterProvider(), clients.telemetry.TracerProvider(), thumbnailService.AMQPHandler)
	if err != nil {
		return services{}, err
	}

	amqpExifService, err := amqphandler.New(config.amqpExif, clients.amqp, clients.telemetry.MeterProvider(), clients.telemetry.TracerProvider(), metadataService.AMQPHandler)
	if err != nil {
		return services{}, err
	}

	searchService := search.New(adapters.filteredStorage, thumbnailService, metadataService, adapters.exclusiveService, clients.telemetry.TracerProvider())

	crudService, err := crud.New(config.crud, adapters.storage, adapters.filteredStorage, rendererService, shareService, webhookService, thumbnailService, metadataService, searchService, eventBus.Push, clients.telemetry.TracerProvider())
	if err != nil {
		return services{}, err
	}

	sanitizerService := sanitizer.New(config.sanitizer, adapters.filteredStorage, adapters.exclusiveService, crudService, eventBus.Push)

	var middlewareService provider.Auth
	if !config.disableAuth {
		middlewareService = newLoginService(clients.telemetry.TracerProvider(), config.basic)
	}

	fibrService := fibr.New(&crudService, rendererService, shareService, webhookService, middlewareService)

	return services{
		eventBus:      eventBus,
		amqpThumbnail: amqpThumbnailService,
		amqpExif:      amqpExifService,
		fibr:          fibrService,
		sanitizer:     sanitizerService,
		renderer:      rendererService,
		webhook:       webhookService,
		share:         shareService,
		thumbnail:     thumbnailService,
		metadata:      metadataService,
		server:        server.New(config.server),
	}, nil
}

func (s services) Start(adapters adapters, doneCtx, endCtx context.Context) {
	go s.amqpThumbnail.Start(doneCtx)
	go s.amqpExif.Start(doneCtx)
	go s.sanitizer.Start(doneCtx)

	go s.webhook.Start(endCtx)
	go s.share.Start(endCtx)

	go s.eventBus.Start(endCtx, adapters.storage, []provider.Renamer{s.thumbnail.Rename, s.metadata.Rename}, s.share.EventConsumer, s.thumbnail.EventConsumer, s.metadata.EventConsumer, s.webhook.EventConsumer)
}

func (s services) Close() {
	<-s.amqpThumbnail.Done()
	<-s.amqpExif.Done()
	<-s.sanitizer.Done()

	<-s.webhook.Done()
	<-s.share.Done()
}

func newLoginService(tracerProvider trace.TracerProvider, basicConfig *basicMemory.Config) provider.Auth {
	basicService, err := basicMemory.New(basicConfig)
	if err != nil {
		slog.LogAttrs(context.Background(), slog.LevelError, "auth memory", slog.Any("error", err))
		os.Exit(1)
	}

	basicProviderProvider := basic.New(basicService, "fibr")
	return authMiddleware.New(basicService, tracerProvider, basicProviderProvider)
}
