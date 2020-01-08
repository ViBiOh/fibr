package filesystem

import (
	"io"
	"os"
	"path"
	"strings"

	"github.com/ViBiOh/fibr/pkg/provider"
)

func getMode(name string) os.FileMode {
	if strings.HasSuffix(name, "/") {
		return 0700
	}

	return 0600
}

func (a app) getFile(filename string) (io.WriteCloser, error) {
	return os.OpenFile(a.getFullPath(filename), os.O_RDWR|os.O_CREATE|os.O_TRUNC, getMode(filename))
}

func convertToItem(dirname string, info os.FileInfo) *provider.StorageItem {
	if info == nil {
		return nil
	}

	name := info.Name()
	pathname := path.Join(dirname, name)

	if strings.EqualFold(dirname, "/") {
		pathname = dirname
	}

	return &provider.StorageItem{
		Name:     name,
		Pathname: pathname,
		IsDir:    info.IsDir(),
		Date:     info.ModTime(),

		Info: info,
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
