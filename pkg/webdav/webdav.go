package webdav

import (
	"context"
	"fmt"
	"os"
	"time"

	absto "github.com/ViBiOh/absto/pkg/model"
	"golang.org/x/net/webdav"
)

var _ webdav.LockSystem = NoLock{}

type NoLock struct{}

func (nl NoLock) Confirm(now time.Time, name0, name1 string, conditions ...webdav.Condition) (release func(), err error) {
	return func() {}, nil
}

func (nl NoLock) Create(now time.Time, details webdav.LockDetails) (token string, err error) {
	return "accepted", nil
}

func (nl NoLock) Refresh(now time.Time, token string, duration time.Duration) (webdav.LockDetails, error) {
	return webdav.LockDetails{}, nil
}

func (nl NoLock) Unlock(now time.Time, token string) error {
	return nil
}

type Filesystem struct {
	storage absto.Storage
}

func NewFilesystem(storage absto.Storage) Filesystem {
	return Filesystem{
		storage: storage,
	}
}

func (f Filesystem) Mkdir(ctx context.Context, name string, perm os.FileMode) error {
	return f.storage.CreateDir(ctx, name)
}

func (f Filesystem) OpenFile(ctx context.Context, name string, flag int, perm os.FileMode) (webdav.File, error) {
	return nil, nil
}

func (f Filesystem) RemoveAll(ctx context.Context, name string) error {
	return f.storage.Remove(ctx, name)
}

func (f Filesystem) Rename(ctx context.Context, oldName, newName string) error {
	return f.storage.Rename(ctx, oldName, newName)
}

func (f Filesystem) Stat(ctx context.Context, name string) (os.FileInfo, error) {
	info, err := f.storage.Info(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("stat: %w", err)
	}

	return info.AsFileInfo(), nil
}
