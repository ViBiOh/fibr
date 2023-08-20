package main

import (
	"github.com/ViBiOh/absto/pkg/absto"
	model "github.com/ViBiOh/absto/pkg/model"
	"github.com/ViBiOh/fibr/pkg/exclusive"
	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/fibr/pkg/storage"
)

type adapters struct {
	storageApp      model.Storage
	filteredStorage model.Storage
	exclusiveApp    exclusive.App
	eventBus        provider.EventBus
}

func newAdapters(config configuration, clients client) (adapters, error) {
	storageApp, err := absto.New(config.absto, clients.telemetry.TracerProvider().Tracer("absto"))
	if err != nil {
		return adapters{}, err
	}

	filteredStorage, err := storage.Get(config.storage, storageApp)
	if err != nil {
		return adapters{}, err
	}

	eventBus, err := provider.NewEventBus(provider.MaxConcurrency, clients.telemetry.MeterProvider(), clients.telemetry.TracerProvider())
	if err != nil {
		return adapters{}, err
	}

	var exclusiveApp exclusive.App
	if clients.redis.Enabled() {
		exclusiveApp = exclusive.New(clients.redis)
	}

	return adapters{
		storageApp:      storageApp,
		filteredStorage: filteredStorage,
		exclusiveApp:    exclusiveApp,
		eventBus:        eventBus,
	}, nil
}
