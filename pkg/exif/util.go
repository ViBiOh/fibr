package exif

import (
	"fmt"
	"path"
	"strings"

	"github.com/ViBiOh/fibr/pkg/provider"
)

func getExifPath(item provider.StorageItem, suffix string) string {
	fullPath := path.Join(provider.MetadataDirectoryName, item.Pathname)
	if item.IsDir {
		return fullPath
	}

	name := strings.TrimSuffix(fullPath, path.Ext(fullPath))

	if len(suffix) == 0 {
		return fmt.Sprintf("%s.json", name)
	}

	return fmt.Sprintf("%s_%s.json", name, suffix)
}
