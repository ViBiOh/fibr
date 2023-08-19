package thumbnail

import (
	"log/slog"

	absto "github.com/ViBiOh/absto/pkg/model"
)

func onCacheError(pathname string, err error) bool {
	if !absto.IsNotExist(err) {
		slog.Error("get info", "err", err, "item", pathname)
	}

	return false
}
