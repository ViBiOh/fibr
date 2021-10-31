package filesystem

import (
	"io"
	"os"
	"strings"

	"github.com/ViBiOh/fibr/pkg/provider"
)

const (
	writeFlags = os.O_RDWR | os.O_CREATE | os.O_TRUNC
)

func checkPathname(pathname string) error {
	if strings.Contains(pathname, "..") {
		return ErrRelativePath
	}

	return nil
}

func (a App) getRelativePath(pathname string) string {
	return strings.TrimPrefix(pathname, a.rootDirectory)
}

func (a App) getFile(filename string, flags int) (*os.File, error) {
	return os.OpenFile(a.path(filename), flags, getMode(filename))
}

func (a App) getWritableFile(filename string) (io.WriteCloser, error) {
	return a.getFile(filename, writeFlags)
}

func getMode(name string) os.FileMode {
	if strings.HasSuffix(name, "/") {
		return 0o700
	}

	return 0o600
}

func convertToItem(pathname string, info os.FileInfo) provider.StorageItem {
	return provider.StorageItem{
		Name:     info.Name(),
		Pathname: pathname,
		IsDir:    info.IsDir(),
		Date:     info.ModTime(),
		Size:     info.Size(),
	}
}

func convertError(err error) error {
	if err == nil {
		return nil
	}

	if os.IsNotExist(err) {
		return provider.ErrNotExist(err)
	}

	return err
}
