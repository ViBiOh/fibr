package exif

import (
	"context"
	"fmt"
	"path/filepath"

	absto "github.com/ViBiOh/absto/pkg/model"
	exas "github.com/ViBiOh/exas/pkg/model"
	"github.com/ViBiOh/fibr/pkg/provider"
)

// CanHaveExif determine if exif can be extracted for given pathname
func (a App) CanHaveExif(item absto.Item) bool {
	return provider.ThumbnailExtensions[item.Extension] && (a.maxSize == 0 || item.Size < a.maxSize || a.directAccess)
}

func getExifPath(item absto.Item) string {
	if item.IsDir {
		return provider.MetadataDirectory(item) + "aggregate.json"
	}

	return fmt.Sprintf("%s%s.json", provider.MetadataDirectory(item), item.ID)
}

func (a App) hasMetadata(ctx context.Context, item absto.Item) bool {
	_, err := a.storageApp.Info(ctx, getExifPath(item))
	return err == nil
}

func (a App) loadExif(ctx context.Context, item absto.Item) (exas.Exif, error) {
	var data exas.Exif
	return data, a.loadMetadata(ctx, item, &data)
}

func (a App) loadAggregate(ctx context.Context, item absto.Item) (provider.Aggregate, error) {
	var data provider.Aggregate
	return data, a.loadMetadata(ctx, item, &data)
}

func (a App) loadMetadata(ctx context.Context, item absto.Item, content interface{}) error {
	return provider.LoadJSON(ctx, a.storageApp, getExifPath(item), content)
}

func (a App) saveMetadata(ctx context.Context, item absto.Item, data interface{}) error {
	filename := getExifPath(item)
	dirname := filepath.Dir(filename)

	if _, err := a.storageApp.Info(ctx, dirname); err != nil {
		if !absto.IsNotExist(err) {
			return fmt.Errorf("unable to check directory existence: %s", err)
		}

		if err = a.storageApp.CreateDir(ctx, dirname); err != nil {
			return fmt.Errorf("unable to create directory: %s", err)
		}
	}

	if err := provider.SaveJSON(ctx, a.storageApp, filename, data); err != nil {
		return fmt.Errorf("unable to save: %s", err)
	}

	if item.IsDir {
		a.increaseAggregate("save")
	} else {
		a.increaseExif("save")
	}

	return nil
}
