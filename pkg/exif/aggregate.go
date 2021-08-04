package exif

import (
	"fmt"
	"path"
	"path/filepath"
	"time"

	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/httputils/v4/pkg/logger"
)

var (
	aggregateRatio = 0.4

	levels = []string{"city", "state", "country"}
)

func (a app) GetAggregateFor(item provider.StorageItem) (provider.Aggregate, error) {
	if !a.enabled() {
		return provider.Aggregate{}, nil
	}

	if !item.IsDir {
		return provider.Aggregate{}, nil
	}

	aggregate, err := a.loadAggregate(item)
	if err != nil {
		return aggregate, fmt.Errorf("unable to load aggregate: %s", err)
	}

	return aggregate, nil
}

func (a app) AggregateFor(item provider.StorageItem) {
	if !a.enabled() {
		return
	}

	if !item.IsDir {
		file, err := a.getDirOf(item)
		if err != nil {
			logger.Error("unable to get directory for `%s`: %s", item.Pathname, err)
			return
		}

		item = file
	}

	for {
		select {
		case <-a.done:
			logger.Warn("Service is going to shutdown, not adding more aggregate to the queue `%s`", item.Pathname)
			return
		case a.aggregateQueue <- item:
			a.increaseMetric("aggregate", "queued")
			return
		default:
			time.Sleep(time.Second)
		}
	}
}

func (a app) processAggregateQueue() {
	for item := range a.aggregateQueue {
		a.decreaseMetric("aggregate", "queued")

		if err := a.computeAndSaveAggregate(item); err != nil {
			logger.Error("unable to compute aggregate for `%s`: %s", item.Pathname, err)
		}
	}
}

func (a app) computeAndSaveAggregate(dir provider.StorageItem) error {
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

		if a.HasExif(item) {
			if itemDate, err := a.getDate(item); err == nil {
				minDate, maxDate = aggregateDate(minDate, maxDate, itemDate)
			}
		}

		if a.HasGeocode(item) {
			data, err := a.loadGeocode(item)
			if err != nil {
				return fmt.Errorf("unable to get geocode: %s", err)
			}

			directoryAggregate.ingest(data)
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("unable to aggregate: %s", err)
	}

	err = a.saveMetadata(dir, aggregateMetadataFilename, provider.Aggregate{
		Location: directoryAggregate.value(),
		Start:    minDate,
		End:      maxDate,
	})
	if err == nil {
		a.increaseMetric("aggregate", "saved")
	}

	return err
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

func (a app) getDirOf(item provider.StorageItem) (provider.StorageItem, error) {
	return a.storageApp.Info(path.Dir(item.Pathname))
}
