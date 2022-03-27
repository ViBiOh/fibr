package exif

import (
	"context"
	"fmt"
	"time"

	absto "github.com/ViBiOh/absto/pkg/model"
	exas "github.com/ViBiOh/exas/pkg/model"
	"github.com/ViBiOh/fibr/pkg/provider"
	"go.opentelemetry.io/otel/trace"
)

var (
	aggregateRatio = 0.4

	levels = []string{"city", "state", "country"}
)

// GetExifFor return exif value for a given item
func (a App) GetExifFor(ctx context.Context, item absto.Item) (exas.Exif, error) {
	if item.IsDir {
		return exas.Exif{}, nil
	}

	if a.tracer != nil {
		var span trace.Span
		ctx, span = a.tracer.Start(ctx, "aggregate")
		defer span.End()
	}

	exif, err := a.loadExif(ctx, item)
	if err != nil && !absto.IsNotExist(err) {
		return exif, fmt.Errorf("unable to load exif: %w", err)
	}

	return exif, nil
}

// SaveExifFor saves given exif for given item
func (a App) SaveExifFor(ctx context.Context, item absto.Item, exif exas.Exif) error {
	return a.saveMetadata(ctx, item, exif)
}

// GetAggregateFor return aggregated value for a given directory
func (a App) GetAggregateFor(ctx context.Context, item absto.Item) (provider.Aggregate, error) {
	if !item.IsDir {
		return provider.Aggregate{}, nil
	}

	if a.tracer != nil {
		var span trace.Span
		ctx, span = a.tracer.Start(ctx, "aggregate")
		defer span.End()
	}

	aggregate, err := a.loadAggregate(ctx, item)
	if err != nil && !absto.IsNotExist(err) {
		return aggregate, fmt.Errorf("unable to load aggregate: %w", err)
	}

	return aggregate, nil
}

func (a App) aggregate(ctx context.Context, item absto.Item) error {
	if !item.IsDir {
		file, err := a.getDirOf(ctx, item)
		if err != nil {
			return fmt.Errorf("unable to get directory: %w", err)
		}

		item = file
	}

	if err := a.computeAndSaveAggregate(ctx, item); err != nil {
		return fmt.Errorf("unable to compute aggregate: %w", err)
	}

	return nil
}

func (a App) computeAndSaveAggregate(ctx context.Context, dir absto.Item) error {
	directoryAggregate := newAggregate()
	var minDate, maxDate time.Time

	err := a.storageApp.Walk(ctx, dir.Pathname, func(item absto.Item) error {
		if item.Pathname == dir.Pathname {
			return nil
		}

		exifData, err := a.loadExif(ctx, item)
		if err != nil {
			if absto.IsNotExist(err) {
				return nil
			}
			return fmt.Errorf("unable load exif data: %w", err)
		}

		if !exifData.Date.IsZero() {
			minDate, maxDate = aggregateDate(minDate, maxDate, exifData.Date)
		}

		if !exifData.Geocode.HasAddress() {
			directoryAggregate.ingest(exifData.Geocode)
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("unable to aggregate: %w", err)
	}

	if len(directoryAggregate) == 0 {
		return nil
	}

	return a.saveMetadata(ctx, dir, provider.Aggregate{
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
