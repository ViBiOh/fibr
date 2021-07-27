package providertest

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"time"

	"github.com/ViBiOh/fibr/pkg/provider"
)

type stubReadCloserSeeker struct {
	bytes.Buffer
}

func (s stubReadCloserSeeker) Seek(_ int64, _ int) (int64, error) {
	return 0, nil
}

func (s stubReadCloserSeeker) Close() error {
	return nil
}

// Storage fakes implementation
type Storage struct {
}

// SetIgnoreFn fakes implementation
func (s Storage) SetIgnoreFn(func(provider.StorageItem) bool) {
	// mock implementation
}

// Info fakes implementation
func (s Storage) Info(pathname string) (provider.StorageItem, error) {
	if strings.HasSuffix(pathname, "error") {
		return provider.StorageItem{}, errors.New("error on info")
	}

	return provider.StorageItem{
		Name: pathname,
	}, nil
}

// WriterTo fakes implementation
func (s Storage) WriterTo(pathname string) (io.WriteCloser, error) {
	if strings.HasSuffix(pathname, "error") {
		return nil, errors.New("error on writer to")
	}

	return nil, nil
}

// ReaderFrom fakes implementation
func (s Storage) ReaderFrom(pathname string) (provider.ReadSeekerCloser, error) {
	if strings.HasSuffix(pathname, "error") {
		return nil, errors.New("error on reader from")
	}

	buffer := stubReadCloserSeeker{}
	if _, err := buffer.WriteString("ok"); err != nil {
		return nil, err
	}

	return &buffer, nil
}

// UpdateDate fakes implementation
func (s Storage) UpdateDate(pathname string, date time.Time) error {
	return nil
}

// List fakes implementation
func (s Storage) List(pathname string) ([]provider.StorageItem, error) {
	if strings.HasSuffix(pathname, "error") {
		return nil, errors.New("error on list")
	}

	return []provider.StorageItem{
		{Name: pathname + "_1"},
		{Name: pathname + "_2"},
	}, nil
}

// Walk fakes implementation
func (s Storage) Walk(pathname string, _ func(provider.StorageItem, error) error) error {
	if strings.HasSuffix(pathname, "error") {
		return errors.New("error on create dir")
	}

	return nil
}

// CreateDir fakes implementation
func (s Storage) CreateDir(pathname string) error {
	if strings.HasSuffix(pathname, "error") {
		return errors.New("error on create dir")
	}

	return nil
}

// Store fakes implementation
func (s Storage) Store(pathname string, content io.ReadCloser) error {
	if strings.HasSuffix(pathname, "error") {
		return errors.New("error on store")
	}

	if err := content.Close(); err != nil {
		return err
	}

	return nil
}

// Rename fakes implementation
func (s Storage) Rename(oldName, _ string) error {
	if oldName == "error" {
		return errors.New("error on rename")
	}

	return nil
}

// Remove fakes implementation
func (s Storage) Remove(pathname string) error {
	if strings.HasSuffix(pathname, "error") {
		return errors.New("error on remove")
	}

	return nil
}
