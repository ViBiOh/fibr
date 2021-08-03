package exif

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/httputils/v4/pkg/logger"
)

var (
	aggregateRatio = 0.4

	levels = []string{"city", "state", "country"}
)

type geocode struct {
	Address map[string]string `json:"address"`
}

type locationAggregate map[string]map[string]int64

func newAggregate() locationAggregate {
	return make(map[string]map[string]int64)
}

func (a *locationAggregate) ingest(geocoding geocode) {
	for _, level := range levels {
		a.inc(level, geocoding.Address[level])
	}
}

func (a locationAggregate) inc(key, value string) {
	if len(value) == 0 {
		return
	}

	if level, ok := a[key]; ok {
		if _, ok := level[value]; ok {
			level[value]++
		} else {
			level[value] = 1
		}
	} else {
		a[key] = map[string]int64{
			value: 1,
		}
	}
}

func (a locationAggregate) value() string {
	if len(a) == 0 {
		return ""
	}

	for _, level := range levels {
		if val := a.valueOf(level); len(val) > 0 {
			return val
		}
	}

	return "Worldwide"
}

func (a locationAggregate) valueOf(key string) string {
	values, ok := a[key]
	if !ok {
		return ""
	}

	var sum int64
	for _, v := range values {
		sum += v
	}

	var names []string
	minSum := int64(float64(sum) * aggregateRatio)

	for k, v := range values {
		if v > minSum {
			names = append(names, k)
		}
	}

	return strings.Join(names, ", ")
}

func (a app) GetAggregateFor(item provider.StorageItem) (provider.Aggregate, error) {
	if !a.enabled() {
		return provider.Aggregate{}, nil
	}

	if !item.IsDir {
		return provider.Aggregate{}, nil
	}

	var aggregate provider.Aggregate

	if err := a.loadMetadata(item, "aggregate", &aggregate); err != nil {
		return aggregate, fmt.Errorf("unable to load metadata: %s", err)
	}

	return aggregate, nil
}

func (a app) AggregateFor(item provider.StorageItem) {
	if !a.enabled() {
		return
	}

	if !item.IsDir {
		return
	}

	select {
	case <-a.done:
		logger.Warn("Service is going to shutdown, not adding more aggregate to the queue `%s`", item.Pathname)
		return
	default:
	}

	a.aggregateQueue <- item
	a.aggregateCounter.WithLabelValues("queued").Inc()
}

func (a app) computeAggregate() {
	for item := range a.aggregateQueue {
		a.aggregateCounter.WithLabelValues("queued").Dec()

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
			data, err := a.getGeocode(item)
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

	err = a.saveMetadata(dir, "aggregate", provider.Aggregate{
		Location: directoryAggregate.value(),
		Start:    minDate,
		End:      maxDate,
	})
	if err == nil {
		a.aggregateCounter.WithLabelValues("saved").Dec()
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

func (a app) getGeocode(item provider.StorageItem) (geocode, error) {
	var data geocode

	reader, err := a.storageApp.ReaderFrom(getExifPath(item, "geocode"))
	if err != nil {
		return geocode{}, fmt.Errorf("unable to read: %s", err)
	}

	if err := json.NewDecoder(reader).Decode(&data); err != nil {
		return geocode{}, fmt.Errorf("unable to decode: %s", err)
	}

	return data, nil
}
