package main

import (
	"github.com/ViBiOh/absto/pkg/absto"
	model "github.com/ViBiOh/absto/pkg/model"
	"github.com/ViBiOh/fibr/pkg/exclusive"
	"github.com/ViBiOh/fibr/pkg/storage"
)

type adapters struct {
	storage          model.Storage
	filteredStorage  model.Storage
	exclusiveService exclusive.Service
}

func newAdapters(config configuration, clients clients) (adapters, error) {
	storageService, err := absto.New(config.absto, clients.telemetry.TracerProvider())
	if err != nil {
		return adapters{}, err
	}

	filteredStorage, err := storage.Get(config.storage, storageService)
	if err != nil {
		return adapters{}, err
	}

	var exclusiveService exclusive.Service
	if clients.redis.Enabled() {
		exclusiveService = exclusive.New(clients.redis)
	}

	return adapters{
		storage:          storageService,
		filteredStorage:  filteredStorage,
		exclusiveService: exclusiveService,
	}, nil
}
