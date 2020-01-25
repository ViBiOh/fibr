package filesystem

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/httputils/v3/pkg/flags"
	"github.com/ViBiOh/httputils/v3/pkg/httperror"
	"github.com/ViBiOh/httputils/v3/pkg/logger"
)

var (
	_ provider.Storage = &app{}

	// ErrRelativePath occurs when path is relative (contains ".."")
	ErrRelativePath = errors.New("pathname contains relatives paths")
)

// Config of package
type Config struct {
	directory *string
	ignore    *string
}

type app struct {
	rootDirectory string
	rootDirname   string
	ignorePattern *regexp.Regexp
}

// Flags adds flags for configuring package
func Flags(fs *flag.FlagSet, prefix string) Config {
	return Config{
		directory: flags.New(prefix, "filesystem").Name("Directory").Default("/data").Label("Path to served directory").ToString(fs),
		ignore:    flags.New(prefix, "filesystem").Name("IgnorePattern").Default("").Label("Ignore pattern when listing files or directory").ToString(fs),
	}
}

// New creates new App from Config
func New(config Config) (provider.Storage, error) {
	rootDirectory := strings.TrimSpace(*config.directory)
	ignore := strings.TrimSpace(*config.ignore)

	if rootDirectory == "" {
		return nil, errors.New("no directory provided")
	}

	info, err := os.Stat(rootDirectory)
	if err != nil {
		return nil, convertError(err)
	}

	if !info.IsDir() {
		return nil, fmt.Errorf("path %s is not a directory", rootDirectory)
	}

	var ignorePattern *regexp.Regexp
	if len(ignore) != 0 {
		pattern, err := regexp.Compile(ignore)
		if err != nil {
			return nil, err
		}

		ignorePattern = pattern
	}

	logger.Info("Serving file from %s", rootDirectory)

	return &app{
		rootDirectory: rootDirectory,
		rootDirname:   info.Name(),
		ignorePattern: ignorePattern,
	}, nil
}

// Name of the storage
func (a app) Name() string {
	return "filesystem"
}

// Root name of the storage
func (a app) Root() string {
	return a.rootDirname
}

// Info provide metadata about given pathname
func (a app) Info(pathname string) (provider.StorageItem, error) {
	if err := a.checkPathname(pathname); err != nil {
		return provider.StorageItem{}, convertError(err)
	}

	fullpath := a.getFullPath(pathname)

	info, err := os.Stat(fullpath)
	if err != nil {
		return provider.StorageItem{}, convertError(err)
	}

	return convertToItem(a.getRelativePath(fullpath), info), nil
}

// WriterTo opens writer for given pathname
func (a app) WriterTo(pathname string) (io.WriteCloser, error) {
	if err := a.checkPathname(pathname); err != nil {
		return nil, convertError(err)
	}

	writer, err := a.getFile(pathname)
	return writer, convertError(err)
}

// ReaderFrom reads content from given pathname
func (a app) ReaderFrom(pathname string) (io.ReadCloser, error) {
	if err := a.checkPathname(pathname); err != nil {
		return nil, convertError(err)
	}

	output, err := os.OpenFile(a.getFullPath(pathname), os.O_RDONLY, getMode(pathname))
	return output, convertError(err)
}

// Serve file for given pathname
func (a app) Serve(w http.ResponseWriter, r *http.Request, pathname string) {
	if err := a.checkPathname(pathname); err != nil {
		httperror.Forbidden(w)
		return
	}

	http.ServeFile(w, r, a.getFullPath(pathname))
}

// List items in the storage
func (a app) List(pathname string) ([]provider.StorageItem, error) {
	if err := a.checkPathname(pathname); err != nil {
		return nil, convertError(err)
	}

	fullpath := a.getFullPath(pathname)

	files, err := ioutil.ReadDir(fullpath)
	if err != nil {
		return nil, convertError(err)
	}

	items := make([]provider.StorageItem, 0)
	for _, item := range files {
		if a.ignorePattern != nil && a.ignorePattern.MatchString(item.Name()) {
			continue
		}

		items = append(items, convertToItem(a.getRelativePath(path.Join(fullpath, item.Name())), item))
	}

	sort.Sort(ByHybridSort(items))

	return items, nil
}

// Walk browses item recursively
func (a app) Walk(pathname string, walkFn func(provider.StorageItem, error) error) error {
	pathname = path.Join(a.rootDirectory, pathname)

	return convertError(filepath.Walk(pathname, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			logger.Error("%s", err)
			return walkFn(provider.StorageItem{}, err)
		}

		return walkFn(convertToItem(a.getRelativePath(path), info), err)
	}))
}

// Create container in storage
func (a app) CreateDir(name string) error {
	if err := a.checkPathname(name); err != nil {
		return convertError(err)
	}

	return convertError(os.MkdirAll(a.getFullPath(name), 0700))
}

// Store file to storage
func (a app) Store(pathname string, content io.ReadCloser) error {
	if err := a.checkPathname(pathname); err != nil {
		return convertError(err)
	}

	storageFile, err := a.getFile(pathname)
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

	copyBuffer := make([]byte, 32*1024)
	if _, err = io.CopyBuffer(storageFile, content, copyBuffer); err != nil {
		return convertError(err)
	}

	return nil
}

// Rename file or directory from storage
func (a app) Rename(oldName, newName string) error {
	if err := a.checkPathname(oldName); err != nil {
		return convertError(err)
	}

	if err := a.checkPathname(newName); err != nil {
		return convertError(err)
	}

	return convertError(os.Rename(a.getFullPath(oldName), a.getFullPath(newName)))
}

// Remove file or directory from storage
func (a app) Remove(pathname string) error {
	if err := a.checkPathname(pathname); err != nil {
		return convertError(err)
	}

	return convertError(os.RemoveAll(a.getFullPath(pathname)))
}
