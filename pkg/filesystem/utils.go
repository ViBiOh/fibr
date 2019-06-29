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

	name := info.Name()

	return &provider.StorageItem{
		Pathname: path.Join(dirname, name),
		Name:     name,
		IsDir:    info.IsDir(),
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
