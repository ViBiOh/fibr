package filesystem

import (
	"io"
	"os"
	"path"

	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/httputils/pkg/errors"
)

func (a app) getFile(filename string) (io.WriteCloser, error) {
	return os.OpenFile(a.getFullPath(filename), os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
}

func convertToItem(dirname string, info os.FileInfo) *provider.StorageItem {
	if info == nil {
		return nil
	}

	pathname := dirname
	name := info.Name()
	isDir := info.IsDir()

	if !isDir {
		pathname = path.Join(dirname, name)
	}

	return &provider.StorageItem{
		Pathname: pathname,
		Name:     name,
		IsDir:    isDir,
		Date:     info.ModTime(),
	}
}

func convertError(err error) error {
	if err == nil {
		return nil
	}

	if os.IsNotExist(err) {
		return provider.ErrNotExist(err)
	}

	return errors.WithStack(err)
}
