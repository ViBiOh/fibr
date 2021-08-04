package exif

import (
	"errors"
	"fmt"
	"time"

	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/httputils/v4/pkg/logger"
)

func (a app) UpdateDateFor(item provider.StorageItem) {
	createDate, err := a.getDate(item)
	if err != nil {
		logger.Error("unable to get date for `%s`: %s", item.Pathname, err)
	}

	if createDate.IsZero() {
		a.dateCounter.WithLabelValues("zero").Inc()
		return
	}

	if item.Date.Equal(createDate) {
		a.dateCounter.WithLabelValues("equal").Inc()
		return
	}

	a.dateCounter.WithLabelValues("updated").Inc()

	if err := a.storageApp.UpdateDate(item.Pathname, createDate); err != nil {
		logger.Error("unable to update date for `%s`: %s", item.Pathname, err)
	}
}

func (a app) getDate(item provider.StorageItem) (time.Time, error) {
	data, err := a.get(item)
	if err != nil {
		return time.Time{}, fmt.Errorf("unable to get exif: %s", err)
	}

	for _, exifDate := range exifDates {
		rawCreateDate, ok := data[exifDate]
		if !ok {
			continue
		}

		createDateStr, ok := rawCreateDate.(string)
		if !ok {
			return time.Time{}, fmt.Errorf("key `%s` is not a string", exifDate)
		}

		createDate, err := parseDate(createDateStr)
		if err == nil {
			return createDate, nil
		}
	}

	return time.Time{}, nil
}

func parseDate(raw string) (time.Time, error) {
	for _, pattern := range datePatterns {
		createDate, err := time.Parse(pattern, raw)
		if err == nil {
			return createDate, nil
		}
	}

	return time.Time{}, errors.New("no matching pattern")
}
