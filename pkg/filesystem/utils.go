package filesystem

import (
	"os"
	"path"

	"github.com/ViBiOh/fibr/pkg/provider"
)

func convertToItem(pathname string, info os.FileInfo) *provider.StorageItem {
	if info == nil {
		return nil
	}

	return &provider.StorageItem{
		Pathname: path.Join(pathname, info.Name()),
		Name:     info.Name(),
		IsDir:    info.IsDir(),
	}
}
