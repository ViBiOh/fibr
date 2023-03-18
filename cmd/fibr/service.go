package main

import (
	"github.com/ViBiOh/absto/pkg/absto"
	model "github.com/ViBiOh/absto/pkg/model"
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
	"github.com/ViBiOh/httputils/v4/pkg/amqphandler"
	"github.com/ViBiOh/httputils/v4/pkg/renderer"
)

type services struct {
	eventBus         provider.EventBus
	storageApp       model.Storage
	webhookApp       *webhook.App
	shareApp         *share.App
	sanitizerApp     sanitizer.App
	fibrApp          fibr.App
	rendererApp      renderer.App
	amqpThumbnailApp amqphandler.App
	amqpExifApp      amqphandler.App
	metadataApp      metadata.App
	thumbnailApp     thumbnail.App
}

func newServices(config configuration, clients client) (services, error) {
	prometheusRegisterer := clients.prometheus.Registerer()

	storageApp, err := absto.New(config.absto, clients.tracer.GetTracer("storage"))
	if err != nil {
		return services{}, err
	}

	filteredStorage, err := storage.Get(config.storage, storageApp)
	if err != nil {
		return services{}, err
	}

	eventBus, err := provider.NewEventBus(provider.MaxConcurrency, prometheusRegisterer, clients.tracer.GetTracer("bus"))
	if err != nil {
		return services{}, err
	}

	thumbnailApp, err := thumbnail.New(config.thumbnail, storageApp, clients.redis, prometheusRegisterer, clients.tracer, clients.amqp)
	if err != nil {
		return services{}, err
	}

	rendererApp, err := renderer.New(config.renderer, content, fibr.FuncMap, clients.tracer.GetTracer("renderer"))
	if err != nil {
		return services{}, err
	}

	var exclusiveApp exclusive.App
	if clients.redis.Enabled() {
		exclusiveApp = exclusive.New(clients.redis)
	}

	metadataApp, err := metadata.New(config.metadata, storageApp, prometheusRegisterer, clients.tracer, clients.amqp, clients.redis, exclusiveApp)
	if err != nil {
		return services{}, err
	}

	webhookApp := webhook.New(config.webhook, storageApp, prometheusRegisterer, clients.redis, rendererApp, thumbnailApp, exclusiveApp)

	shareApp, err := share.New(config.share, storageApp, clients.redis, exclusiveApp)
	if err != nil {
		return services{}, err
	}

	amqpThumbnailApp, err := amqphandler.New(config.amqpThumbnail, clients.amqp, clients.tracer.GetTracer("amqp_handler_thumbnail"), thumbnailApp.AMQPHandler)
	if err != nil {
		return services{}, err
	}

	amqpExifApp, err := amqphandler.New(config.amqpExif, clients.amqp, clients.tracer.GetTracer("amqp_handler_exif"), metadataApp.AMQPHandler)
	if err != nil {
		return services{}, err
	}

	searchApp := search.New(filteredStorage, thumbnailApp, metadataApp, exclusiveApp, clients.tracer.GetTracer("search"))

	crudApp, err := crud.New(config.crud, storageApp, filteredStorage, rendererApp, shareApp, webhookApp, thumbnailApp, metadataApp, searchApp, eventBus.Push, clients.tracer.GetTracer("crud"))
	if err != nil {
		return services{}, err
	}

	sanitizerApp := sanitizer.New(config.sanitizer, filteredStorage, exclusiveApp, crudApp, eventBus.Push)

	var middlewareApp provider.Auth
	if !*config.disableAuth {
		middlewareApp = newLoginApp(clients.tracer.GetTracer("auth"), config.basic)
	}

	fibrApp := fibr.New(&crudApp, rendererApp, shareApp, webhookApp, middlewareApp)

	return services{
		storageApp:       storageApp,
		amqpThumbnailApp: amqpThumbnailApp,
		amqpExifApp:      amqpExifApp,
		fibrApp:          fibrApp,
		sanitizerApp:     sanitizerApp,
		rendererApp:      rendererApp,
		eventBus:         eventBus,
		webhookApp:       webhookApp,
		shareApp:         shareApp,
		thumbnailApp:     thumbnailApp,
		metadataApp:      metadataApp,
	}, nil
}
