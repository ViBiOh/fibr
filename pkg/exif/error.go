package exif

import (
	"errors"

	absto "github.com/ViBiOh/absto/pkg/model"
	"github.com/ViBiOh/httputils/v4/pkg/logger"
)

func onListError(item absto.Item, err error) bool {
	if !absto.IsNotExist(err) && !errors.Is(err, errInvalidItemType) {
		logger.WithField("item", item.Pathname).Error("load exif: %s", item.Pathname, err)
	}

	return true
}
