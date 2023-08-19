package metadata

import (
	"context"
	"fmt"
	"log/slog"

	absto "github.com/ViBiOh/absto/pkg/model"
	"github.com/ViBiOh/fibr/pkg/exclusive"
	"github.com/ViBiOh/fibr/pkg/provider"
)

func (a App) Update(ctx context.Context, item absto.Item, opts ...provider.MetadataAction) (provider.Metadata, error) {
	var output provider.Metadata

	return output, a.exclusiveApp.Execute(ctx, "fibr:mutex:"+item.ID, exclusive.Duration, func(ctx context.Context) error {
		var err error

		metadata, err := a.GetMetadataFor(ctx, item)
		if err != nil && !absto.IsNotExist(err) {
			slog.Error("load metadata", "err", err, "item", item.Pathname)
		}

		for _, opt := range opts {
			metadata = opt(metadata)
		}

		if err = a.exifCacheApp.EvictOnSuccess(ctx, item, a.saveMetadata(ctx, item, metadata)); err != nil {
			return fmt.Errorf("save metadata: %w", err)
		}

		output = metadata

		return nil
	})
}
