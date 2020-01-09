package thumbnail

import (
	"context"
	"fmt"
	"io"
	"path"
	"strings"
	"time"

	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/httputils/v3/pkg/logger"
)

const (
	defaultTimeout = time.Second * 30
)

func getCtx(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx, defaultTimeout)
}

// CanHaveThumbnail determine if thumbnail can be generated for given pathname
func CanHaveThumbnail(item *provider.StorageItem) bool {
	return item.IsImage() || item.IsPdf()
}

func safeWrite(w io.Writer, content string) {
	if _, err := io.WriteString(w, content); err != nil {
		logger.Error("%s", err)
	}
}

func getThumbnailPath(item *provider.StorageItem) string {
	fullPath := path.Join(provider.MetadataDirectoryName, item.Pathname)
	return fmt.Sprintf("%s.jpg", strings.TrimSuffix(fullPath, path.Ext(fullPath)))
}
