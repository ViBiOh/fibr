package thumbnail

import (
	"context"
	"fmt"
	"path"
	"strings"
	"time"

	"github.com/ViBiOh/fibr/pkg/provider"
)

const (
	defaultTimeout = time.Second * 30
)

func getCtx(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx, defaultTimeout)
}

// CanHaveThumbnail determine if thumbnail can be generated for given pathname
func CanHaveThumbnail(item provider.StorageItem) bool {
	return item.IsImage() || item.IsPdf() || item.IsVideo()
}

func getThumbnailPath(item provider.StorageItem) string {
	fullPath := path.Join(provider.MetadataDirectoryName, item.Pathname)
	return fmt.Sprintf("%s.jpg", strings.TrimSuffix(fullPath, path.Ext(fullPath)))
}
