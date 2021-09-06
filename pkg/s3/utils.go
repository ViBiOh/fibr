package s3

import (
	"strings"

	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/minio/minio-go/v7"
)

func getPath(pathname string) string {
	return strings.TrimPrefix(pathname, "/")
}

func convertToItem(pathname string, info minio.ObjectInfo) provider.StorageItem {
	return provider.StorageItem{
		Name:     strings.TrimSuffix(info.Key, "/"),
		Pathname: pathname,
		IsDir:    strings.HasSuffix(info.Key, "/"),
		Date:     info.LastModified,
		Size:     info.Size,
	}
}
