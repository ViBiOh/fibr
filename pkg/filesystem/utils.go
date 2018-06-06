package filesystem

import (
	"os"
	"path"
	"strings"

	"github.com/ViBiOh/fibr/pkg/provider"
)

func convertToItem(pathname string, info os.FileInfo) *provider.StorageItem {
	if info == nil {
		return nil
	}

	return &provider.StorageItem{
		Pathname: path.Join(pathname, info.Name()),
		Name:     strings.TrimSpace(info.Name()),
		IsDir:    info.IsDir(),
	}
}
