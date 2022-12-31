package metadata

import (
	"context"
	"fmt"

	absto "github.com/ViBiOh/absto/pkg/model"
	"github.com/ViBiOh/fibr/pkg/exclusive"
	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/httputils/v4/pkg/logger"
)

func (a App) Update(ctx context.Context, item absto.Item, opts ...provider.MetadataAction) (provider.Metadata, error) {
	var metadata provider.Metadata

	return metadata, a.exclusiveApp.Execute(ctx, "fibr:mutex:"+item.ID, exclusive.Duration, func(ctx context.Context) error {
		var err error

		metadata, err := a.GetMetadataFor(ctx, item)
		if err != nil && !absto.IsNotExist(err) {
			logger.WithField("item", item.Pathname).Error("load metadata: %s", err)
		}

		for _, opt := range opts {
			metadata = opt(metadata)
		}

		if err = a.exifCacheApp.EvictOnSuccess(ctx, item, a.saveMetadata(ctx, item, metadata)); err != nil {
			return fmt.Errorf("save metadata: %w", err)
		}

		return nil
	})
}
