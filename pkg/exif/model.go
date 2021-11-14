package exif

import (
	"strings"
	"time"
)

type exif struct {
	Date    time.Time `json:"date"`
	Geocode geocode   `json:"geocode"`
}

func (e exif) IsZero() bool {
	return e.Date.IsZero() && len(e.Geocode.Address) == 0 && len(e.Geocode.Latitude) == 0
}

type geocode struct {
	Address   map[string]string `json:"address"`
	Latitude  string            `json:"lat"`
	Longitude string            `json:"lon"`
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
		level[value]++
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
