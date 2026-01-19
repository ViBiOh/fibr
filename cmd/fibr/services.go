package main

import (
	"context"
	"embed"
	"errors"
	"log/slog"
	"os"

	basicMemory "github.com/ViBiOh/auth/v3/pkg/store/memory"
	"github.com/ViBiOh/fibr/pkg/cookie"
	"github.com/ViBiOh/fibr/pkg/crud"
	"github.com/ViBiOh/fibr/pkg/fibr"
	"github.com/ViBiOh/fibr/pkg/metadata"
	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/fibr/pkg/push"
	"github.com/ViBiOh/fibr/pkg/sanitizer"
	"github.com/ViBiOh/fibr/pkg/search"
	"github.com/ViBiOh/fibr/pkg/share"
	"github.com/ViBiOh/fibr/pkg/thumbnail"
	"github.com/ViBiOh/fibr/pkg/webhook"
	"github.com/ViBiOh/httputils/v4/pkg/amqphandler"
	"github.com/ViBiOh/httputils/v4/pkg/owasp"
	"github.com/ViBiOh/httputils/v4/pkg/renderer"
	"github.com/ViBiOh/httputils/v4/pkg/server"
)

//go:embed templates static
var content embed.FS

type services struct {
	server   *server.Server
	owasp    owasp.Service
	renderer *renderer.Service

	fibr          fibr.Service
	eventBus      provider.EventBus
	webhook       *webhook.Service
	share         *share.Service
	amqpThumbnail *amqphandler.Service
	amqpExif      *amqphandler.Service
	sanitizer     sanitizer.Service
	metadata      *metadata.Service
	thumbnail     thumbnail.Service
}

func newServices(ctx context.Context, config configuration, clients clients, adapters adapters) (services, error) {
	var output services
	var err error

	output.server = server.New(config.server)
	output.owasp = owasp.New(config.owasp)

	output.eventBus, err = provider.NewEventBus(provider.MaxConcurrency, clients.telemetry.MeterProvider(), clients.telemetry.TracerProvider())
	if err != nil {
		return output, err
	}

	output.thumbnail, err = thumbnail.New(ctx, config.thumbnail, adapters.storage, clients.redis, clients.telemetry.MeterProvider(), clients.telemetry.TracerProvider(), clients.amqp)
	if err != nil {
		return output, err
	}

	output.renderer, err = renderer.New(ctx, config.renderer, content, fibr.FuncMap, clients.telemetry.MeterProvider(), clients.telemetry.TracerProvider())
	if err != nil {
		return output, err
	}

	output.metadata, err = metadata.New(ctx, config.metadata, adapters.storage, clients.telemetry.MeterProvider(), clients.telemetry.TracerProvider(), clients.amqp, clients.redis, adapters.exclusiveService)
	if err != nil {
		return output, err
	}

	pushService, err := push.New(config.push, adapters.storage, adapters.exclusiveService)
	if err != nil && !errors.Is(err, push.ErrNoConfig) {
		return output, err
	}

	output.webhook = webhook.New(config.webhook, adapters.storage, clients.telemetry.MeterProvider(), clients.redis, output.renderer, pushService, output.thumbnail, adapters.exclusiveService)

	output.share, err = share.New(config.share, clients.telemetry.TracerProvider(), adapters.storage, clients.redis, adapters.exclusiveService)
	if err != nil {
		return output, err
	}

	output.amqpThumbnail, err = amqphandler.New(config.amqpThumbnail, clients.amqp, clients.telemetry.MeterProvider(), clients.telemetry.TracerProvider(), output.thumbnail.AMQPHandler)
	if err != nil {
		return output, err
	}

	output.amqpExif, err = amqphandler.New(config.amqpExif, clients.amqp, clients.telemetry.MeterProvider(), clients.telemetry.TracerProvider(), output.metadata.AMQPHandler)
	if err != nil {
		return output, err
	}

	searchService := search.New(adapters.filteredStorage, output.thumbnail, output.metadata, adapters.exclusiveService, clients.telemetry.TracerProvider())

	crudService, err := crud.New(config.crud, adapters.storage, adapters.filteredStorage, output.renderer, output.share, output.webhook, output.thumbnail, output.metadata, searchService, pushService, output.eventBus.Push, clients.telemetry.TracerProvider())
	if err != nil {
		return output, err
	}

	output.sanitizer = sanitizer.New(config.sanitizer, adapters.filteredStorage, adapters.exclusiveService, crudService, output.eventBus.Push)

	var middlewareService provider.Auth
	if !config.disableAuth {
		middlewareService = newLoginService(config.basic)
	}

	output.fibr = fibr.New(crudService, output.renderer, output.share, output.webhook, middlewareService, cookie.New[cookie.BasicContent](config.cookie))

	return output, nil
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

func newLoginService(basicConfig *basicMemory.Config) provider.Auth {
	basicService, err := basicMemory.New(basicConfig)
	if err != nil {
		slog.LogAttrs(context.Background(), slog.LevelError, "auth memory", slog.Any("error", err))
		os.Exit(1)
	}

	return basicService
}
