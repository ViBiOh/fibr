package exif

import (
	"context"
	"fmt"
	"time"

	absto "github.com/ViBiOh/absto/pkg/model"
	exas "github.com/ViBiOh/exas/pkg/model"
	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/fibr/pkg/version"
	"github.com/ViBiOh/httputils/v4/pkg/cache"
	"github.com/ViBiOh/httputils/v4/pkg/tracer"
)

var (
	cacheDuration  = time.Hour * 96
	aggregateRatio = 0.4

	levels = []string{"city", "state", "country"}
)

func redisKey(itemID string) string {
	return version.Redis("exif:" + itemID)
}

func (a App) GetExifFor(ctx context.Context, item absto.Item) (exas.Exif, error) {
	if item.IsDir {
		return exas.Exif{}, nil
	}

	ctx, end := tracer.StartSpan(ctx, a.tracer, "get_exif")
	defer end()

	return cache.Retrieve(ctx, a.redisClient, func(ctx context.Context) (exas.Exif, error) {
		exif, err := a.loadExif(ctx, item)
		if err != nil && !absto.IsNotExist(err) {
			return exif, fmt.Errorf("load exif: %w", err)
		}

		return exif, nil
	}, cacheDuration, redisKey(item.ID))
}

func (a App) GetAggregateFor(ctx context.Context, item absto.Item) (provider.Aggregate, error) {
	if !item.IsDir {
		return provider.Aggregate{}, nil
	}

	ctx, end := tracer.StartSpan(ctx, a.tracer, "aggregate")
	defer end()

	return cache.Retrieve(ctx, a.redisClient, func(ctx context.Context) (provider.Aggregate, error) {
		aggregate, err := a.loadAggregate(ctx, item)
		if err != nil && !absto.IsNotExist(err) {
			return aggregate, fmt.Errorf("load aggregate: %w", err)
		}

		return aggregate, nil
	}, cacheDuration, redisKey(item.ID))
}

func (a App) SaveExifFor(ctx context.Context, item absto.Item, exif exas.Exif) error {
	return cache.EvictOnSuccess(ctx, a.redisClient, redisKey(item.ID), a.saveMetadata(ctx, item, exif))
}

func (a App) SaveAggregateFor(ctx context.Context, item absto.Item, aggregate provider.Aggregate) error {
	return cache.EvictOnSuccess(ctx, a.redisClient, redisKey(item.ID), a.saveMetadata(ctx, item, aggregate))
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

		exifData, err := a.GetExifFor(ctx, item)
		if err != nil {
			if absto.IsNotExist(err) {
				return nil
			}
			return fmt.Errorf("unable load exif data: %w", err)
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

	if len(directoryAggregate) == 0 {
		return nil
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
