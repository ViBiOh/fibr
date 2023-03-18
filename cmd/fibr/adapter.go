package main

import (
	"github.com/ViBiOh/absto/pkg/absto"
	model "github.com/ViBiOh/absto/pkg/model"
	"github.com/ViBiOh/fibr/pkg/exclusive"
	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/fibr/pkg/storage"
	"github.com/prometheus/client_golang/prometheus"
)

type adapters struct {
	prometheusRegisterer prometheus.Registerer
	storageApp           model.Storage
	filteredStorage      model.Storage
	exclusiveApp         exclusive.App
	eventBus             provider.EventBus
}

func newAdapters(config configuration, clients client) (adapters, error) {
	prometheusRegisterer := clients.prometheus.Registerer()

	storageApp, err := absto.New(config.absto, clients.tracer.GetTracer("storage"))
	if err != nil {
		return adapters{}, err
	}

	filteredStorage, err := storage.Get(config.storage, storageApp)
	if err != nil {
		return adapters{}, err
	}

	eventBus, err := provider.NewEventBus(provider.MaxConcurrency, prometheusRegisterer, clients.tracer.GetTracer("bus"))
	if err != nil {
		return adapters{}, err
	}

	var exclusiveApp exclusive.App
	if clients.redis.Enabled() {
		exclusiveApp = exclusive.New(clients.redis)
	}

	return adapters{
		prometheusRegisterer: prometheusRegisterer,
		storageApp:           storageApp,
		filteredStorage:      filteredStorage,
		exclusiveApp:         exclusiveApp,
		eventBus:             eventBus,
	}, nil
}
