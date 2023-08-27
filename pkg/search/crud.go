package search

import (
	"context"
	"fmt"
	"path/filepath"

	absto "github.com/ViBiOh/absto/pkg/model"
	"github.com/ViBiOh/fibr/pkg/exclusive"
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
	if item.IsDir() {
		return provider.MetadataDirectory(item) + "searches.json"
	}

	return fmt.Sprintf("%s%s.json", provider.MetadataDirectory(item), item.ID)
}

func (s Service) List(ctx context.Context, item absto.Item) (Searches, error) {
	searches, err := s.load(ctx, item)
	if err != nil {
		return nil, fmt.Errorf("load: %w", err)
	}

	return searches, nil
}

func (s Service) Get(ctx context.Context, item absto.Item, name string) (provider.Search, error) {
	searches, err := s.List(ctx, item)
	if err != nil {
		return provider.Search{}, fmt.Errorf("list: %w", err)
	}

	return searches[name], nil
}

func (s Service) Add(ctx context.Context, item absto.Item, search provider.Search) error {
	return s.update(ctx, item, DoAdd(search))
}

func (s Service) Delete(ctx context.Context, item absto.Item, name string) error {
	return s.update(ctx, item, DoRemove(name))
}

func (s Service) update(ctx context.Context, item absto.Item, opts ...SearchesOption) error {
	return s.exclusive.Execute(ctx, "fibr:mutex:"+item.ID, exclusive.Duration, func(ctx context.Context) error {
		searches, err := s.load(ctx, item)
		if err != nil {
			return fmt.Errorf("load: %w", err)
		}

		for _, opt := range opts {
			searches = opt(searches)
		}

		if err = s.save(ctx, item, searches); err != nil {
			return fmt.Errorf("save: %w", err)
		}

		return nil
	})
}

func (s Service) load(ctx context.Context, item absto.Item) (Searches, error) {
	output, err := provider.LoadJSON[Searches](ctx, s.storage, path(item))
	if err != nil {
		if !absto.IsNotExist(err) {
			return nil, err
		}

		return make(map[string]provider.Search), nil
	}

	return output, nil
}

func (s Service) save(ctx context.Context, item absto.Item, content Searches) error {
	filename := path(item)
	dirname := filepath.Dir(filename)

	if _, err := s.storage.Stat(ctx, dirname); err != nil {
		if !absto.IsNotExist(err) {
			return fmt.Errorf("check directory existence: %w", err)
		}

		if err = s.storage.Mkdir(ctx, dirname, absto.DirectoryPerm); err != nil {
			return fmt.Errorf("create directory: %w", err)
		}
	}

	if err := provider.SaveJSON(ctx, s.storage, filename, content); err != nil {
		return fmt.Errorf("save: %w", err)
	}

	return nil
}
