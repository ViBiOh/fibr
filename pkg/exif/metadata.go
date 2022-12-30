package exif

import (
	"context"
	"fmt"

	absto "github.com/ViBiOh/absto/pkg/model"
	exas "github.com/ViBiOh/exas/pkg/model"
	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/httputils/v4/pkg/logger"
)

type MetadataOption func(provider.Metadata) provider.Metadata

func WithExif(exif exas.Exif) MetadataOption {
	return func(instance provider.Metadata) provider.Metadata {
		instance.Exif = exif

		return instance
	}
}

func WithDescription(description string) MetadataOption {
	return func(instance provider.Metadata) provider.Metadata {
		instance.Description = description

		return instance
	}
}

func (a App) update(ctx context.Context, item absto.Item, opts ...MetadataOption) (provider.Metadata, error) {
	var metadata provider.Metadata

	return metadata, a.exclusiveApp.Execute(ctx, "fibr:mutex:"+item.ID, func(ctx context.Context) error {
		var err error

		metadata, err := a.GetMetadataFor(ctx, item)
		if err != nil && !absto.IsNotExist(err) {
			logger.WithField("item", item.Pathname).Error("load exif: %s", err)
		}

		for _, opt := range opts {
			metadata = opt(metadata)
		}

		if err = a.exifCacheApp.EvictOnSuccess(ctx, item, a.saveMetadata(ctx, item, metadata)); err != nil {
			return fmt.Errorf("save exif: %w", err)
		}

		return nil
	})
}

func (a App) UpdateDescription(ctx context.Context, item absto.Item, description string) error {
	_, err := a.update(ctx, item, WithDescription(description))
	return err
}
