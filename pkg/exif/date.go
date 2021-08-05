package exif

import (
	"errors"
	"fmt"
	"time"

	"github.com/ViBiOh/fibr/pkg/provider"
)

func (a app) updateDate(item provider.StorageItem) error {
	createDate, err := a.getDate(item)
	if err != nil {
		return fmt.Errorf("unable to get date: %s", err)
	}

	if createDate.IsZero() {
		return nil
	}

	if item.Date.Equal(createDate) {
		return nil
	}

	if err := a.storageApp.UpdateDate(item.Pathname, createDate); err != nil {
		return fmt.Errorf("unable to update date: %s", err)
	}

	return nil
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
