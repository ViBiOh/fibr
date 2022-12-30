package metadata

import (
	"context"
	"fmt"
	"time"

	absto "github.com/ViBiOh/absto/pkg/model"
	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/fibr/pkg/version"
	"github.com/ViBiOh/httputils/v4/pkg/tracer"
)

var (
	cacheDuration  = time.Hour * 96
	aggregateRatio = 0.4

	levels = []string{"city", "state", "country"}
)

func redisKey(item absto.Item) string {
	return version.Redis("exif:" + item.ID)
}

func (a App) GetMetadataFor(ctx context.Context, item absto.Item) (provider.Metadata, error) {
	if item.IsDir {
		return provider.Metadata{}, nil
	}

	ctx, end := tracer.StartSpan(ctx, a.tracer, "get_metadata")
	defer end()

	return a.exifCacheApp.Get(ctx, item)
}

func (a App) GetAllMetadataFor(ctx context.Context, items ...absto.Item) (map[string]provider.Metadata, error) {
	ctx, end := tracer.StartSpan(ctx, a.tracer, "get_all_metadata")
	defer end()

	exifs, err := a.exifCacheApp.List(ctx, onListError, items...)
	if err != nil {
		return nil, fmt.Errorf("list: %w", err)
	}

	output := make(map[string]provider.Metadata, len(items))
	exifsLen := len(exifs)

	for index, item := range items {
		if index < exifsLen {
			output[item.ID] = exifs[index]
		}
	}

	return output, nil
}

func (a App) GetAggregateFor(ctx context.Context, item absto.Item) (provider.Aggregate, error) {
	if !item.IsDir {
		return provider.Aggregate{}, nil
	}

	ctx, end := tracer.StartSpan(ctx, a.tracer, "get_aggregate")
	defer end()

	return a.aggregateCacheApp.Get(ctx, item)
}

func (a App) GetAllAggregateFor(ctx context.Context, items ...absto.Item) (map[string]provider.Aggregate, error) {
	ctx, end := tracer.StartSpan(ctx, a.tracer, "get_all_aggregate")
	defer end()

	exifs, err := a.aggregateCacheApp.List(ctx, onListError, items...)
	if err != nil {
		return nil, fmt.Errorf("list: %w", err)
	}

	output := make(map[string]provider.Aggregate, len(items))
	exifsLen := len(exifs)

	for index, item := range items {
		if index < exifsLen {
			output[item.ID] = exifs[index]
		}
	}

	return output, nil
}

func (a App) SaveAggregateFor(ctx context.Context, item absto.Item, aggregate provider.Aggregate) error {
	return a.aggregateCacheApp.EvictOnSuccess(ctx, item, a.saveMetadata(ctx, item, aggregate))
}

func (a App) aggregate(ctx context.Context, item absto.Item) error {
	if !item.IsDir {
		file, err := a.getDirOf(ctx, item)
		if err != nil {
			return fmt.Errorf("get directory: %w", err)
		}

		item = file
	}

	if err := a.computeAndSaveAggregate(ctx, item); err != nil {
		return fmt.Errorf("compute aggregate: %w", err)
	}

	return nil
}

func (a App) computeAndSaveAggregate(ctx context.Context, dir absto.Item) error {
	directoryAggregate := newAggregate()
	var minDate, maxDate time.Time

	previousAggregate, _ := a.GetAggregateFor(ctx, dir)

	err := a.storageApp.Walk(ctx, dir.Pathname, func(item absto.Item) error {
		if item.Pathname == dir.Pathname {
			return nil
		}

		exifData, err := a.GetMetadataFor(ctx, item)
		if err != nil {
			if absto.IsNotExist(err) {
				return nil
			}

			return fmt.Errorf("load exif data: %w", err)
		}

		if !exifData.Date.IsZero() {
			minDate, maxDate = aggregateDate(minDate, maxDate, exifData.Date)
		}

		if exifData.Geocode.HasAddress() {
			directoryAggregate.ingest(exifData.Geocode)
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("aggregate: %w", err)
	}

	return a.SaveAggregateFor(ctx, dir, provider.Aggregate{
		Cover:    previousAggregate.Cover,
		Location: directoryAggregate.value(),
		Start:    minDate,
		End:      maxDate,
	})
}

func aggregateDate(min, max, current time.Time) (time.Time, time.Time) {
	if min.IsZero() || current.Before(min) {
		min = current
	}

	if max.IsZero() || current.After(max) {
		max = current
	}

	return min, max
}

func (a App) getDirOf(ctx context.Context, item absto.Item) (absto.Item, error) {
	return a.storageApp.Info(ctx, item.Dir())
}
