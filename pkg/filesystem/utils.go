package filesystem

import (
	"io"
	"os"
	"path"
	"strings"

	"github.com/ViBiOh/fibr/pkg/provider"
)

func checkPathname(pathname string) error {
	if strings.Contains(pathname, "..") {
		return ErrRelativePath
	}

	return nil
}

func (a app) getFullPath(pathname string) string {
	return path.Join(a.rootDirectory, pathname)
}

func (a app) getRelativePath(pathname string) string {
	return strings.TrimPrefix(pathname, a.rootDirectory)
}

func (a app) getWritableFile(filename string) (io.WriteCloser, error) {
	return os.OpenFile(a.getFullPath(filename), os.O_RDWR|os.O_CREATE|os.O_TRUNC, getMode(filename))
}

func getMode(name string) os.FileMode {
	if strings.HasSuffix(name, "/") {
		return 0700
	}

	return 0600
}

func convertToItem(pathname string, info os.FileInfo) provider.StorageItem {
	return provider.StorageItem{
		Name:     info.Name(),
		Pathname: pathname,
		IsDir:    info.IsDir(),
		Date:     info.ModTime(),
		Info:     info,
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
