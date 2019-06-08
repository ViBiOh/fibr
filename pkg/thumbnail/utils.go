package thumbnail

import (
	"context"
	"fmt"
	"io"
	"path"
	"strings"

	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/httputils/pkg/errors"
	"github.com/ViBiOh/httputils/pkg/logger"
)

func getCtx(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx, defaultTimeout)
}

// CanHaveThumbnail determine if thumbnail can be generated for given pathname
func CanHaveThumbnail(pathname string) bool {
	extension := strings.ToLower(path.Ext(pathname))

	return provider.ImageExtensions[extension] || provider.PdfExtensions[extension]
}

func safeWrite(w io.Writer, content string) {
	if _, err := io.WriteString(w, content); err != nil {
		logger.Error("%#v", errors.WithStack(err))
	}
}

func getThumbnailPath(pathname string) string {
	fullPath := path.Join(provider.MetadataDirectoryName, pathname)

	return fmt.Sprintf("%s.jpg", strings.TrimSuffix(fullPath, path.Ext(fullPath)))
}
