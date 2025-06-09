package metadata

import (
	"context"
	"fmt"
	"log/slog"

	absto "github.com/ViBiOh/absto/pkg/model"
	"github.com/ViBiOh/fibr/pkg/exclusive"
	"github.com/ViBiOh/fibr/pkg/provider"
)

func (s *Service) Update(ctx context.Context, item absto.Item, opts ...provider.MetadataAction) (provider.Metadata, error) {
	var output provider.Metadata

	return output, s.exclusive.Execute(ctx, "fibr:mutex:"+item.ID, exclusive.Duration, func(ctx context.Context) error {
		var err error

		metadata, err := s.GetMetadataFor(ctx, item)
		if err != nil && !absto.IsNotExist(err) {
			slog.LogAttrs(ctx, slog.LevelError, "load metadata", slog.String("item", item.Pathname), slog.Any("error", err))
		}

		for _, opt := range opts {
			metadata = opt(metadata)
		}

		if err = s.exifCache.EvictOnSuccess(ctx, item, s.saveMetadata(ctx, item, metadata)); err != nil {
			return fmt.Errorf("save metadata: %w", err)
		}

		output = metadata

		return nil
	})
}
