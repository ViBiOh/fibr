package filesystem

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/httputils/v4/pkg/flags"
	"github.com/ViBiOh/httputils/v4/pkg/logger"
)

var (
	// ErrRelativePath occurs when path is relative (contains ".."")
	ErrRelativePath = errors.New("pathname contains relatives paths")
)

// Config of package
type Config struct {
	directory *string
}

// App of the package
type App struct {
	ignoreFn func(provider.StorageItem) bool

	rootDirectory string
	rootDirname   string
}

// Flags adds flags for configuring package
func Flags(fs *flag.FlagSet, prefix string) Config {
	return Config{
		directory: flags.New(prefix, "filesystem", "Directory").Default("/data", nil).Label("Path to served directory").ToString(fs),
	}
}

// New creates new App from Config
func New(config Config) (App, error) {
	rootDirectory := strings.TrimSuffix(strings.TrimSpace(*config.directory), "/")

	if len(rootDirectory) == 0 {
		return App{}, errors.New("no directory provided")
	}

	info, err := os.Stat(rootDirectory)
	if err != nil {
		return App{}, convertError(err)
	}

	if !info.IsDir() {
		return App{}, fmt.Errorf("path %s is not a directory", rootDirectory)
	}

	logger.Info("Serving file from %s", rootDirectory)

	return App{
		rootDirectory: rootDirectory,
		rootDirname:   info.Name(),
	}, nil
}

func (a *App) path(pathname string) string {
	return path.Join(a.rootDirectory, pathname)
}

// WithIgnoreFn create a new App with given ignoreFn
func (a *App) WithIgnoreFn(ignoreFn func(provider.StorageItem) bool) {
	a.ignoreFn = ignoreFn
}

// Info provide metadata about given pathname
func (a *App) Info(pathname string) (provider.StorageItem, error) {
	if err := checkPathname(pathname); err != nil {
		return provider.StorageItem{}, convertError(err)
	}

	fullpath := a.path(pathname)

	info, err := os.Stat(fullpath)
	if err != nil {
		return provider.StorageItem{}, convertError(err)
	}

	return convertToItem(a.getRelativePath(fullpath), info), nil
}

// List items in the storage
func (a *App) List(pathname string) ([]provider.StorageItem, error) {
	if err := checkPathname(pathname); err != nil {
		return nil, convertError(err)
	}

	fullpath := a.path(pathname)

	files, err := os.ReadDir(fullpath)
	if err != nil {
		return nil, convertError(err)
	}

	items := make([]provider.StorageItem, 0)
	for _, file := range files {
		fileInfo, err := file.Info()
		if err != nil {
			return nil, fmt.Errorf("unable to read file metadata: %s", err)
		}

		item := convertToItem(a.getRelativePath(path.Join(fullpath, file.Name())), fileInfo)
		if a.ignoreFn != nil && a.ignoreFn(item) {
			continue
		}

		items = append(items, item)
	}

	sort.Sort(ByHybridSort(items))

	return items, nil
}

// WriterTo opens writer for given pathname
func (a *App) WriterTo(pathname string) (io.WriteCloser, error) {
	if err := checkPathname(pathname); err != nil {
		return nil, convertError(err)
	}

	writer, err := a.getWritableFile(pathname)
	return writer, convertError(err)
}

// ReaderFrom reads content from given pathname
func (a *App) ReaderFrom(pathname string) (provider.ReadSeekerCloser, error) {
	if err := checkPathname(pathname); err != nil {
		return nil, convertError(err)
	}

	output, err := a.getFile(pathname, os.O_RDONLY)
	return output, convertError(err)
}

// UpdateDate update date from given value
func (a *App) UpdateDate(pathname string, date time.Time) error {
	if err := checkPathname(pathname); err != nil {
		return convertError(err)
	}

	return convertError(os.Chtimes(a.path(pathname), date, date))
}

// Walk browses item recursively
func (a *App) Walk(pathname string, walkFn func(provider.StorageItem, error) error) error {
	pathname = path.Join(a.rootDirectory, pathname)

	return convertError(filepath.Walk(pathname, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			logger.Error("%s", err)
			return walkFn(provider.StorageItem{}, err)
		}

		item := convertToItem(a.getRelativePath(path), info)
		if a.ignoreFn != nil && a.ignoreFn(item) {
			if item.IsDir {
				return filepath.SkipDir
			}
			return nil
		}

		return walkFn(item, err)
	}))
}

// CreateDir container in storage
func (a *App) CreateDir(name string) error {
	if err := checkPathname(name); err != nil {
		return convertError(err)
	}

	return convertError(os.MkdirAll(a.path(name), 0700))
}

// Store file to storage
func (a *App) Store(pathname string, content io.ReadCloser) error {
	defer func() {
		if err := content.Close(); err != nil {
			logger.Error("unable to close content: %s", err)
		}
	}()

	if err := checkPathname(pathname); err != nil {
		return convertError(err)
	}

	storageFile, err := a.getWritableFile(pathname)
	if storageFile != nil {
		defer func() {
			if err := storageFile.Close(); err != nil {
				logger.Error("unable to close stored file: %s", err)
			}
		}()
	}

	if err != nil {
		return convertError(err)
	}

	buffer := provider.BufferPool.Get().(*bytes.Buffer)
	defer provider.BufferPool.Put(buffer)

	if _, err = io.CopyBuffer(storageFile, content, buffer.Bytes()); err != nil {
		return convertError(err)
	}

	return nil
}

// Rename file or directory from storage
func (a *App) Rename(oldName, newName string) error {
	if err := checkPathname(oldName); err != nil {
		return convertError(err)
	}

	if err := checkPathname(newName); err != nil {
		return convertError(err)
	}

	if err := a.CreateDir(filepath.Dir(newName)); err != nil {
		return convertError(err)
	}

	return convertError(os.Rename(a.path(oldName), a.path(newName)))
}

// Remove file or directory from storage
func (a *App) Remove(pathname string) error {
	if err := checkPathname(pathname); err != nil {
		return convertError(err)
	}

	return convertError(os.RemoveAll(a.path(pathname)))
}
