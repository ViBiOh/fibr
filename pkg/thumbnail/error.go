package thumbnail

import (
	absto "github.com/ViBiOh/absto/pkg/model"
	"github.com/ViBiOh/httputils/v4/pkg/logger"
)

func onCacheError(pathname string, err error) bool {
	if !absto.IsNotExist(err) {
		logger.WithField("item", pathname).Error("get info: %s", pathname, err)
	}

	return false
}
