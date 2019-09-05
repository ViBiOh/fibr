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
	"sort"
	"strings"

	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/httputils/pkg/httperror"
	"github.com/ViBiOh/httputils/pkg/logger"
	"github.com/ViBiOh/httputils/pkg/tools"
)

var (
	_ provider.Storage = &app{}

	// ErrRelativePath occurs when path is relative (contains ".."")
	ErrRelativePath = errors.New("pathname contains relatives paths")
)

// Config of package
type Config struct {
	directory *string
}

type app struct {
	rootDirectory string
	rootDirname   string
}

// Flags adds flags for configuring package
func Flags(fs *flag.FlagSet, prefix string) Config {
	return Config{
		directory: fs.String(tools.ToCamel(fmt.Sprintf("%sDirectory", prefix)), "/data", "[filesystem] Path to served directory"),
	}
}

// New creates new App from Config
func New(config Config) (provider.Storage, error) {
	rootDirectory := strings.TrimSpace(*config.directory)

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

	logger.Info("Serving file from %s", rootDirectory)

	return &app{
		rootDirectory: rootDirectory,
		rootDirname:   info.Name(),
	}, nil
}

func (a app) checkPathname(pathname string) error {
	if strings.Contains(pathname, "..") {
		return ErrRelativePath
	}

	return nil
}

func (a app) getFullPath(pathname string) string {
	return path.Join(a.rootDirectory, pathname)
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
func (a app) Info(pathname string) (*provider.StorageItem, error) {
	if err := a.checkPathname(pathname); err != nil {
		return nil, convertError(err)
	}

	info, err := os.Stat(a.getFullPath(pathname))
	if err != nil {
		return nil, convertError(err)
	}

	return convertToItem(path.Dir(pathname), info), nil
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
func (a app) List(pathname string) ([]*provider.StorageItem, error) {
	if err := a.checkPathname(pathname); err != nil {
		return nil, convertError(err)
	}

	files, err := ioutil.ReadDir(a.getFullPath(pathname))
	if err != nil {
		return nil, convertError(err)
	}

	items := make([]*provider.StorageItem, len(files))
	for index, item := range files {
		items[index] = convertToItem(pathname, item)
	}

	sort.Sort(ByHybridSort(items))

	return items, nil
}

// Walk browses item recursively
func (a app) Walk(walkFn func(*provider.StorageItem, error) error) error {
	return convertError(filepath.Walk(a.rootDirectory, func(pathname string, info os.FileInfo, err error) error {
		return walkFn(convertToItem(path.Dir(strings.TrimPrefix(pathname, a.rootDirectory)), info), err)
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
				logger.Error("%#v", err)
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
