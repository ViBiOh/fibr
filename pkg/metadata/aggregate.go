package metadata

import (
	"context"
	"fmt"
	"time"

	absto "github.com/ViBiOh/absto/pkg/model"
	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/fibr/pkg/version"
	"github.com/ViBiOh/httputils/v4/pkg/telemetry"
)

var (
	aggregateRatio = 0.4

	levels = []string{"city", "state", "country"}
)

func redisKey(item absto.Item) string {
	return version.Redis("exif:" + item.ID)
}

func (s *Service) GetMetadataFor(ctx context.Context, item absto.Item) (metadata provider.Metadata, err error) {
	if item.IsDir() {
		return provider.Metadata{}, nil
	}

	ctx, end := telemetry.StartSpan(ctx, s.tracer, "get_metadata")
	defer end(&err)

	return s.exifCache.Get(ctx, item)
}

func (s *Service) GetAllMetadataFor(ctx context.Context, items ...absto.Item) (map[string]provider.Metadata, error) {
	var err error

	ctx, end := telemetry.StartSpan(ctx, s.tracer, "get_all_metadata")
	defer end(&err)

	items = provider.KeepOnlyFile(items)

	exifs, err := s.exifCache.List(ctx, provider.IgnoreNotExistsErr[absto.Item], items...)
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

func (s *Service) GetAggregateFor(ctx context.Context, item absto.Item) (aggregate provider.Aggregate, err error) {
	if !item.IsDir() {
		return provider.Aggregate{}, nil
	}

	ctx, end := telemetry.StartSpan(ctx, s.tracer, "get_aggregate")
	defer end(&err)

	return s.aggregateCache.Get(ctx, item)
}

func (s *Service) GetAllAggregateFor(ctx context.Context, items ...absto.Item) (map[string]provider.Aggregate, error) {
	var err error

	ctx, end := telemetry.StartSpan(ctx, s.tracer, "get_all_aggregate")
	defer end(&err)

	exifs, err := s.aggregateCache.List(ctx, provider.IgnoreNotExistsErr[absto.Item], provider.KeepOnlyDir(items)...)
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

func (s *Service) SaveAggregateFor(ctx context.Context, item absto.Item, aggregate provider.Aggregate) error {
	return s.aggregateCache.EvictOnSuccess(ctx, item, s.saveMetadata(ctx, item, aggregate))
}

func (s *Service) aggregate(ctx context.Context, item absto.Item) error {
	if !item.IsDir() {
		file, err := s.getDirOf(ctx, item)
		if err != nil {
			return fmt.Errorf("get directory: %w", err)
		}

		item = file
	}

	if err := s.computeAndSaveAggregate(ctx, item); err != nil {
		return fmt.Errorf("compute aggregate: %w", err)
	}

	return nil
}

func (s *Service) computeAndSaveAggregate(ctx context.Context, dir absto.Item) error {
	directoryAggregate := newAggregate()
	var minDate, maxDate time.Time

	previousAggregate, _ := s.GetAggregateFor(ctx, dir)

	err := s.storage.Walk(ctx, dir.Pathname, func(item absto.Item) error {
		if item.Pathname == dir.Pathname {
			return nil
		}

		exifData, err := s.GetMetadataFor(ctx, item)
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

	return s.SaveAggregateFor(ctx, dir, provider.Aggregate{
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

func (s *Service) getDirOf(ctx context.Context, item absto.Item) (absto.Item, error) {
	return s.storage.Stat(ctx, item.Dir())
}
