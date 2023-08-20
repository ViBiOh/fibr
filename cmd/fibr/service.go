package main

import (
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
	webhookApp       *webhook.App
	shareApp         *share.App
	sanitizerApp     sanitizer.App
	fibrApp          fibr.App
	rendererApp      *renderer.App
	amqpThumbnailApp *amqphandler.App
	amqpExifApp      *amqphandler.App
	metadataApp      metadata.App
	thumbnailApp     thumbnail.App
}

func newServices(config configuration, clients client, adapters adapters) (services, error) {
	thumbnailApp, err := thumbnail.New(config.thumbnail, adapters.storageApp, clients.redis, clients.telemetry.MeterProvider(), clients.telemetry.TracerProvider(), clients.amqp)
	if err != nil {
		return services{}, err
	}

	rendererApp, err := renderer.New(config.renderer, content, fibr.FuncMap, clients.telemetry.MeterProvider(), clients.telemetry.TracerProvider())
	if err != nil {
		return services{}, err
	}

	metadataApp, err := metadata.New(config.metadata, adapters.storageApp, clients.telemetry.MeterProvider(), clients.telemetry.TracerProvider(), clients.amqp, clients.redis, adapters.exclusiveApp)
	if err != nil {
		return services{}, err
	}

	webhookApp := webhook.New(config.webhook, adapters.storageApp, clients.telemetry.MeterProvider(), clients.redis, rendererApp, thumbnailApp, adapters.exclusiveApp)

	shareApp, err := share.New(config.share, adapters.storageApp, clients.redis, adapters.exclusiveApp)
	if err != nil {
		return services{}, err
	}

	amqpThumbnailApp, err := amqphandler.New(config.amqpThumbnail, clients.amqp, clients.telemetry.MeterProvider(), clients.telemetry.TracerProvider(), thumbnailApp.AMQPHandler)
	if err != nil {
		return services{}, err
	}

	amqpExifApp, err := amqphandler.New(config.amqpExif, clients.amqp, clients.telemetry.MeterProvider(), clients.telemetry.TracerProvider(), metadataApp.AMQPHandler)
	if err != nil {
		return services{}, err
	}

	searchApp := search.New(adapters.filteredStorage, thumbnailApp, metadataApp, adapters.exclusiveApp, clients.telemetry.TracerProvider())

	crudApp, err := crud.New(config.crud, adapters.storageApp, adapters.filteredStorage, rendererApp, shareApp, webhookApp, thumbnailApp, metadataApp, searchApp, adapters.eventBus.Push, clients.telemetry.TracerProvider())
	if err != nil {
		return services{}, err
	}

	sanitizerApp := sanitizer.New(config.sanitizer, adapters.filteredStorage, adapters.exclusiveApp, crudApp, adapters.eventBus.Push)

	var middlewareApp provider.Auth
	if !*config.disableAuth {
		middlewareApp = newLoginApp(clients.telemetry.TracerProvider(), config.basic)
	}

	fibrApp := fibr.New(&crudApp, rendererApp, shareApp, webhookApp, middlewareApp)

	return services{
		amqpThumbnailApp: amqpThumbnailApp,
		amqpExifApp:      amqpExifApp,
		fibrApp:          fibrApp,
		sanitizerApp:     sanitizerApp,
		rendererApp:      rendererApp,
		webhookApp:       webhookApp,
		shareApp:         shareApp,
		thumbnailApp:     thumbnailApp,
		metadataApp:      metadataApp,
	}, nil
}
