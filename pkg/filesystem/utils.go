package filesystem

import (
	"os"

	"github.com/ViBiOh/fibr/pkg/provider"
)

func convertToItem(info os.FileInfo) *provider.StorageItem {
	if info == nil {
		return nil
	}

	return &provider.StorageItem{
		Name:  info.Name(),
		IsDir: info.IsDir(),
	}
}
