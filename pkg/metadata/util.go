package metadata

import (
	"context"
	"fmt"
	"path/filepath"

	absto "github.com/ViBiOh/absto/pkg/model"
	"github.com/ViBiOh/fibr/pkg/provider"
)

func (s *Service) CanHaveExif(item absto.Item) bool {
	return provider.ThumbnailExtensions[item.Extension] && (s.maxSize == 0 || item.Size() < s.maxSize || s.directAccess)
}

func Path(item absto.Item) string {
	if item.IsDir() {
		return provider.MetadataDirectory(item) + "aggregate.json"
	}

	return fmt.Sprintf("%s%s.json", provider.MetadataDirectory(item), item.ID)
}

func (s *Service) hasMetadata(ctx context.Context, item absto.Item) bool {
	if item.IsDir() {
		_, err := s.GetAggregateFor(ctx, item)
		return err == nil
	}

	data, err := s.GetMetadataFor(ctx, item)
	if err != nil {
		return false
	}

	return data.HasData()
}

func (s *Service) loadExif(ctx context.Context, item absto.Item) (provider.Metadata, error) {
	return loadMetadata[provider.Metadata](ctx, s.storage, item)
}

func (s *Service) loadAggregate(ctx context.Context, item absto.Item) (provider.Aggregate, error) {
	return loadMetadata[provider.Aggregate](ctx, s.storage, item)
}

func loadMetadata[T any](ctx context.Context, storageService absto.Storage, item absto.Item) (T, error) {
	return provider.LoadJSON[T](ctx, storageService, Path(item))
}

func (s *Service) saveMetadata(ctx context.Context, item absto.Item, data any) error {
	filename := Path(item)
	dirname := filepath.Dir(filename)

	if _, err := s.storage.Stat(ctx, dirname); err != nil {
		if !absto.IsNotExist(err) {
			return fmt.Errorf("check directory existence: %w", err)
		}

		if err = s.storage.Mkdir(ctx, dirname, absto.DirectoryPerm); err != nil {
			return fmt.Errorf("create directory `%s`: %w", dirname, err)
		}
	}

	if err := provider.SaveJSON(ctx, s.storage, filename, data); err != nil {
		return fmt.Errorf("save: %w", err)
	}

	if item.IsDir() {
		s.increaseAggregate(ctx, "save")
	} else {
		s.increaseExif(ctx, "save")
	}

	return nil
}
