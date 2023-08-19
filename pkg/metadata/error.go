package metadata

import (
	"errors"
	"log/slog"

	absto "github.com/ViBiOh/absto/pkg/model"
)

func onListError(item absto.Item, err error) bool {
	if !absto.IsNotExist(err) && !errors.Is(err, errInvalidItemType) {
		slog.Error("load exif", "err", err, "item", item.Pathname)
	}

	return true
}
