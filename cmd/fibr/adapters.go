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
	var output adapters
	var err error

	output.storage, err = absto.New(config.absto, clients.telemetry.TracerProvider())
	if err != nil {
		return output, err
	}

	output.filteredStorage, err = storage.Get(config.storage, output.storage)
	if err != nil {
		return output, err
	}

	if clients.redis.Enabled() {
		output.exclusiveService = exclusive.New(clients.redis)
	}

	return output, nil
}
