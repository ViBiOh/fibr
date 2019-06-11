package filesystem

import (
	"io"
	"os"
	"path"

	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/httputils/pkg/errors"
)

func getFile(filename string) (io.WriteCloser, error) {
	return os.OpenFile(filename, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
}

func convertToItem(pathname string, info os.FileInfo) *provider.StorageItem {
	if info == nil {
		return nil
	}

	return &provider.StorageItem{
		Pathname: path.Join(pathname, info.Name()),
		Name:     info.Name(),
		IsDir:    info.IsDir(),
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
