package exif

import (
	"fmt"
	"time"

	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/httputils/v4/pkg/logger"
)

func (a App) updateDate(item provider.StorageItem, data map[string]interface{}) error {
	if data == nil {
		var err error
		if data, err = a.get(item); err != nil {
			return fmt.Errorf("unable to get exif: %s", err)
		}
	}

	createDate := getDate(data)

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

func getDate(data map[string]interface{}) time.Time {
	rawDate, ok := data["date"]
	if !ok {
		return time.Time{}
	}

	date, ok := rawDate.(time.Time)
	if !ok {
		logger.Error("date has invalid format `%s`", data["date"])
	}

	return date
}
