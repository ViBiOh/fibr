package s3

import (
	"path"
	"strings"

	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/minio/minio-go/v7"
)

func getPath(pathname string) string {
	return strings.TrimPrefix(pathname, "/")
}

func convertToItem(pathname string, info minio.ObjectInfo) provider.StorageItem {
	return provider.StorageItem{
		Name:     path.Base(info.Key),
		Pathname: info.Key,
		IsDir:    strings.HasSuffix(info.Key, "/"),
		Date:     info.LastModified,
		Size:     info.Size,
	}
}

func convertError(err error) error {
	if err == nil {
		return err
	}

	if strings.Contains(err.Error(), "The specified key does not exist") {
		return provider.ErrNotExist(err)
	}

	return err
}
