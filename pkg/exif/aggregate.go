package exif

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/ViBiOh/fibr/pkg/provider"
)

var (
	aggregateRatio = 0.4

	levels = []string{"city", "state", "country"}
)

// GetAggregateFor return aggregated value for a given directory
func (a App) GetAggregateFor(item provider.StorageItem) (provider.Aggregate, error) {
	if !a.enabled() {
		return provider.Aggregate{}, nil
	}

	if !item.IsDir {
		return provider.Aggregate{}, nil
	}

	aggregate, err := a.loadAggregate(item)
	if err != nil && !provider.IsNotExist(err) {
		return aggregate, fmt.Errorf("unable to load aggregate: %s", err)
	}

	return aggregate, nil
}

func (a App) aggregate(item provider.StorageItem) error {
	if !item.IsDir {
		file, err := a.getDirOf(item)
		if err != nil {
			return fmt.Errorf("unable to get directory: %s", err)
		}

		item = file
	}

	if err := a.computeAndSaveAggregate(item); err != nil {
		return fmt.Errorf("unable to compute aggregate: %s", err)
	}

	return nil
}

func (a App) computeAndSaveAggregate(dir provider.StorageItem) error {
	directoryAggregate := newAggregate()
	var minDate, maxDate time.Time

	err := a.storageApp.Walk(dir.Pathname, func(item provider.StorageItem, err error) error {
		if err != nil {
			return err
		}

		if item.Pathname == dir.Pathname {
			return nil
		}

		if item.IsDir {
			return filepath.SkipDir
		}

		exifData, err := a.loadExif(item)
		if err != nil {
			return fmt.Errorf("unable load exif data: %s", err)
		}

		if !exifData.Date.IsZero() {
			minDate, maxDate = aggregateDate(minDate, maxDate, exifData.Date)
		}

		if !exifData.Geocode.IsZero() {
			directoryAggregate.ingest(exifData.Geocode)
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("unable to aggregate: %s", err)
	}

	return a.saveMetadata(dir, aggregateMetadataFilename, provider.Aggregate{
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

func (a App) getDirOf(item provider.StorageItem) (provider.StorageItem, error) {
	return a.storageApp.Info(item.Dir())
}
