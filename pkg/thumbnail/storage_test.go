package thumbnail

import (
	"bytes"
	"errors"
	"io"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/ViBiOh/fibr/pkg/provider"
)

type stubStorage struct {
	root string
}

func (ss stubStorage) Root() string {
	return ss.root
}

func (ss stubStorage) Info(pathname string) (provider.StorageItem, error) {
	if strings.HasSuffix(pathname, "error") {
		return provider.StorageItem{}, errors.New("error on info")
	}

	return provider.StorageItem{
		Name: pathname,
	}, nil
}

func (ss stubStorage) WriterTo(pathname string) (io.WriteCloser, error) {
	if strings.HasSuffix(pathname, "error") {
		return nil, errors.New("error on writer to")
	}

	return nil, nil
}

func (ss stubStorage) ReaderFrom(pathname string) (io.ReadCloser, error) {
	if strings.HasSuffix(pathname, "error") {
		return nil, errors.New("error on reader from")
	}

	buffer := bytes.Buffer{}
	buffer.WriteString("ok")

	return ioutil.NopCloser(&buffer), nil
}

func (ss stubStorage) Serve(w http.ResponseWriter, r *http.Request, pathname string) {
	if r.URL.Path == "error" || strings.HasSuffix(pathname, "error") {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (ss stubStorage) List(pathname string) ([]provider.StorageItem, error) {
	if strings.HasSuffix(pathname, "error") {
		return nil, errors.New("error on list")
	}

	return []provider.StorageItem{
		{Name: pathname + "_1"},
		{Name: pathname + "_2"},
	}, nil
}

func (ss stubStorage) Walk(pathname string, walkFn func(provider.StorageItem, error) error) error {
	if strings.HasSuffix(pathname, "error") {
		return errors.New("error on create dir")
	}

	return nil
}

func (ss stubStorage) CreateDir(pathname string) error {
	if strings.HasSuffix(pathname, "error") {
		return errors.New("error on create dir")
	}

	return nil
}

func (ss stubStorage) Store(pathname string, content io.ReadCloser) error {
	if strings.HasSuffix(pathname, "error") {
		return errors.New("error on store")
	}

	if err := content.Close(); err != nil {
		return err
	}

	return nil
}

func (ss stubStorage) Rename(oldName, newName string) error {
	if oldName == "error" {
		return errors.New("error on rename")
	}

	return nil
}

func (ss stubStorage) Remove(pathname string) error {
	if strings.HasSuffix(pathname, "error") {
		return errors.New("error on remove")
	}

	return nil
}
