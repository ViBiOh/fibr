package exif

import (
	"context"
	"fmt"
	"path/filepath"

	absto "github.com/ViBiOh/absto/pkg/model"
	exas "github.com/ViBiOh/exas/pkg/model"
	"github.com/ViBiOh/fibr/pkg/provider"
)

func (a App) CanHaveExif(item absto.Item) bool {
	return provider.ThumbnailExtensions[item.Extension] && (a.maxSize == 0 || item.Size < a.maxSize || a.directAccess)
}

func Path(item absto.Item) string {
	if item.IsDir {
		return provider.MetadataDirectory(item) + "aggregate.json"
	}

	return fmt.Sprintf("%s%s.json", provider.MetadataDirectory(item), item.ID)
}

func (a App) hasMetadata(ctx context.Context, item absto.Item) bool {
	if item.IsDir {
		_, err := a.storageApp.Info(ctx, Path(item))
		return err == nil
	}

	data, err := a.loadExif(ctx, item)
	if err != nil {
		return false
	}

	return data.HasData()
}

func (a App) loadExif(ctx context.Context, item absto.Item) (exas.Exif, error) {
	var data exas.Exif
	return data, a.loadMetadata(ctx, item, &data)
}

func (a App) loadAggregate(ctx context.Context, item absto.Item) (provider.Aggregate, error) {
	var data provider.Aggregate
	return data, a.loadMetadata(ctx, item, &data)
}

func (a App) loadMetadata(ctx context.Context, item absto.Item, content any) error {
	return provider.LoadJSON(ctx, a.storageApp, Path(item), content)
}

func (a App) saveMetadata(ctx context.Context, item absto.Item, data any) error {
	filename := Path(item)
	dirname := filepath.Dir(filename)

	if _, err := a.storageApp.Info(ctx, dirname); err != nil {
		if !absto.IsNotExist(err) {
			return fmt.Errorf("check directory existence: %w", err)
		}

		if err = a.storageApp.CreateDir(ctx, dirname); err != nil {
			return fmt.Errorf("create directory: %w", err)
		}
	}

	if err := provider.SaveJSON(ctx, a.storageApp, filename, data); err != nil {
		return fmt.Errorf("save: %w", err)
	}

	if item.IsDir {
		a.increaseAggregate("save")
	} else {
		a.increaseExif("save")
	}

	return nil
}
