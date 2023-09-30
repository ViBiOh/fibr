package main

import (
	"context"

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
)

type services struct {
	webhook       *webhook.Service
	share         *share.Service
	sanitizer     sanitizer.Service
	fibr          fibr.Service
	renderer      *renderer.Service
	amqpThumbnail *amqphandler.Service
	amqpExif      *amqphandler.Service
	metadata      metadata.Service
	thumbnail     thumbnail.Service
}

func newServices(ctx context.Context, config configuration, clients client, adapters adapters) (services, error) {
	thumbnailService, err := thumbnail.New(config.thumbnail, adapters.storage, clients.redis, clients.telemetry.MeterProvider(), clients.telemetry.TracerProvider(), clients.amqp)
	if err != nil {
		return services{}, err
	}

	rendererService, err := renderer.New(config.renderer, content, fibr.FuncMap, clients.telemetry.MeterProvider(), clients.telemetry.TracerProvider())
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

	crudService, err := crud.New(config.crud, adapters.storage, adapters.filteredStorage, rendererService, shareService, webhookService, thumbnailService, metadataService, searchService, adapters.eventBus.Push, clients.telemetry.TracerProvider())
	if err != nil {
		return services{}, err
	}

	sanitizerService := sanitizer.New(config.sanitizer, adapters.filteredStorage, adapters.exclusiveService, crudService, adapters.eventBus.Push)

	var middlewareService provider.Auth
	if !config.disableAuth {
		middlewareService = newLoginService(clients.telemetry.TracerProvider(), config.basic)
	}

	fibrService := fibr.New(&crudService, rendererService, shareService, webhookService, middlewareService)

	return services{
		amqpThumbnail: amqpThumbnailService,
		amqpExif:      amqpExifService,
		fibr:          fibrService,
		sanitizer:     sanitizerService,
		renderer:      rendererService,
		webhook:       webhookService,
		share:         shareService,
		thumbnail:     thumbnailService,
		metadata:      metadataService,
	}, nil
}
