package exif

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/ViBiOh/fibr/pkg/provider"
)

var (
	aggregateRatio = 0.4
)

type geocode struct {
	Address map[string]string `json:"address"`
}

type aggregate map[string]map[string]int64

func newAggregate() aggregate {
	return make(map[string]map[string]int64)
}

func (a *aggregate) ingest(geocoding geocode) {
	a.inc("country", geocoding.Address["country"])
	a.inc("state", geocoding.Address["state"])
	a.inc("city", geocoding.Address["city"])
	a.inc("neighbourhood", geocoding.Address["neighbourhood"])
}

func (a aggregate) inc(key, value string) {
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

func (a aggregate) value() string {
	if len(a) == 0 {
		return ""
	}

	if neighbourhood := a.valueOf("neighbourhood"); len(neighbourhood) > 0 {
		return neighbourhood
	}

	if city := a.valueOf("city"); len(city) > 0 {
		return city
	}

	if state := a.valueOf("state"); len(state) > 0 {
		return state
	}

	if country := a.valueOf("country"); len(country) > 0 {
		return country
	}

	return "Worldwide"
}

func (a aggregate) valueOf(key string) string {
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

func (a app) GeolocationFor(dir provider.StorageItem) (string, error) {
	if !a.enabled() {
		return "", nil
	}

	if !dir.IsDir {
		return "", nil
	}

	directoryAggregate := newAggregate()

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

		if !a.HasGeocode(item) {
			return nil
		}

		data, err := a.getGeocode(item)
		if err != nil {
			return fmt.Errorf("unable to get geocode: %s", err)
		}

		directoryAggregate.ingest(data)

		return nil
	})

	if err != nil {
		return "", fmt.Errorf("unable to aggregate geocode: %s", err)
	}

	return directoryAggregate.value(), nil
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
