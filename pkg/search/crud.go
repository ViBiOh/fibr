package search

import (
	"context"
	"fmt"
	"path/filepath"

	absto "github.com/ViBiOh/absto/pkg/model"
	"github.com/ViBiOh/fibr/pkg/provider"
)

type Searches = map[string]provider.Search

type SearchesOption func(Searches) Searches

func DoAdd(search provider.Search) SearchesOption {
	return func(instance Searches) Searches {
		instance[search.Name] = search

		return instance
	}
}

func DoRemove(name string) SearchesOption {
	return func(instance Searches) Searches {
		delete(instance, name)

		return instance
	}
}

func path(item absto.Item) string {
	if item.IsDir {
		return provider.MetadataDirectory(item) + "searches.json"
	}

	return fmt.Sprintf("%s%s.json", provider.MetadataDirectory(item), item.ID)
}

func (a App) List(ctx context.Context, item absto.Item) (Searches, error) {
	searches, err := a.load(ctx, item)
	if err != nil {
		return nil, fmt.Errorf("load: %w", err)
	}

	return searches, nil
}

func (a App) Get(ctx context.Context, item absto.Item, name string) (provider.Search, error) {
	searches, err := a.List(ctx, item)
	if err != nil {
		return provider.Search{}, fmt.Errorf("list: %w", err)
	}

	return searches[name], nil
}

func (a App) Add(ctx context.Context, item absto.Item, search provider.Search) error {
	return a.update(ctx, item, DoAdd(search))
}

func (a App) Delete(ctx context.Context, item absto.Item, name string) error {
	return a.update(ctx, item, DoRemove(name))
}

func (a App) update(ctx context.Context, item absto.Item, opts ...SearchesOption) error {
	return provider.Exclusive(ctx, a.amqpClient, a.amqpExclusiveRoutingKey, func(ctx context.Context) error {
		searches, err := a.load(ctx, item)
		if err != nil {
			return fmt.Errorf("load: %w", err)
		}

		for _, opt := range opts {
			searches = opt(searches)
		}

		if err = a.save(ctx, item, searches); err != nil {
			return fmt.Errorf("save: %w", err)
		}

		return nil
	})
}

func (a App) load(ctx context.Context, item absto.Item) (Searches, error) {
	output, err := provider.LoadJSON[Searches](ctx, a.storageApp, path(item))
	if err != nil {
		if !absto.IsNotExist(err) {
			return nil, err
		}

		return make(map[string]provider.Search), nil
	}

	return output, nil
}

func (a App) save(ctx context.Context, item absto.Item, content Searches) error {
	filename := path(item)
	dirname := filepath.Dir(filename)

	if _, err := a.storageApp.Info(ctx, dirname); err != nil {
		if !absto.IsNotExist(err) {
			return fmt.Errorf("check directory existence: %w", err)
		}

		if err = a.storageApp.CreateDir(ctx, dirname); err != nil {
			return fmt.Errorf("create directory: %w", err)
		}
	}

	if err := provider.SaveJSON(ctx, a.storageApp, filename, content); err != nil {
		return fmt.Errorf("save: %w", err)
	}

	return nil
}
