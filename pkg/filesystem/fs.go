package filesystem

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/httputils/pkg/tools"
)

var (
	// ErrRelativePath occurs when path is relative (contains ".."")
	ErrRelativePath = errors.New(`pathname contains relatives paths`)

	// ErrOutsidePath occurs when path is not under served directory
	ErrOutsidePath = errors.New(`pathname does not belong to served directory`)
)

// App stores informations
type App struct {
	rootDirectory string
	rootDirname   string
}

// NewApp creates new App from Flags' config
func NewApp(config map[string]*string) (*App, error) {
	rootDirectory := *config[`directory`]

	if rootDirectory == `` {
		return nil, nil
	}

	info, err := os.Stat(rootDirectory)
	if err != nil {
		return nil, err
	}

	if !info.IsDir() {
		return nil, fmt.Errorf(`Directory %s is unreachable`, rootDirectory)
	}

	app := &App{
		rootDirectory: rootDirectory,
		rootDirname:   info.Name(),
	}

	log.Printf(`Serving file from %s`, rootDirectory)

	return app, nil
}

// Flags adds flags for given prefix
func Flags(prefix string) map[string]*string {
	return map[string]*string{
		`directory`: flag.String(tools.ToCamel(fmt.Sprintf(`%sDirectory`, prefix)), `/data`, `[filesystem] Path to served directory`),
	}
}

func getFile(filename string) (io.WriteCloser, error) {
	return os.OpenFile(filename, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
}

func (a App) checkPathname(pathname string) error {
	if strings.Contains(pathname, `..`) {
		return ErrRelativePath
	}

	return nil
}

func (a App) getFullPath(pathname string) string {
	return path.Join(a.rootDirectory, pathname)
}

// Name of the storage
func (a App) Name() string {
	return `filesystem`
}

// Info provide metadata about given pathname
func (a App) Info(pathname string) (*provider.StorageItem, error) {
	if err := a.checkPathname(pathname); err != nil {
		return nil, err
	}

	info, err := os.Stat(a.getFullPath(pathname))
	if err != nil {
		return nil, err
	}

	return convertToItem(info), nil
}

// Read content of given pathname
func (a App) Read(pathname string) (io.ReadCloser, error) {
	if err := a.checkPathname(pathname); err != nil {
		return nil, err
	}

	return os.OpenFile(a.getFullPath(pathname), os.O_RDONLY, 0600)
}

// Open writer for given pathname
func (a App) Open(pathname string) (io.WriteCloser, error) {
	if err := a.checkPathname(pathname); err != nil {
		return nil, err
	}

	return getFile(a.getFullPath(pathname))
}

// List item in the storage
func (a App) List(pathname string) ([]*provider.StorageItem, error) {
	if err := a.checkPathname(pathname); err != nil {
		return nil, err
	}

	files, err := ioutil.ReadDir(a.getFullPath(pathname))
	if err != nil {
		return nil, err
	}

	items := make([]*provider.StorageItem, len(files))
	for index, item := range files {
		items[index] = convertToItem(item)
	}

	return items, nil
}

// Walk browse item recursively
func (a App) Walk(walkFn func(string, *provider.StorageItem, error) error) error {
	return filepath.Walk(a.rootDirectory, func(path string, info os.FileInfo, err error) error {
		return walkFn(strings.TrimPrefix(path, a.rootDirectory), convertToItem(info), err)
	})
}

// Create container in storage
func (a App) Create(name string) error {
	if err := a.checkPathname(name); err != nil {
		return err
	}

	return os.MkdirAll(name, 0700)
}

// Upload file to storage
func (a App) Upload(pathname string, content io.ReadCloser) error {
	if err := a.checkPathname(pathname); err != nil {
		return err
	}

	storageFile, err := getFile(a.getFullPath(pathname))
	if storageFile != nil {
		defer func() {
			if err := storageFile.Close(); err != nil {
				log.Printf(`Error while closing file: %v`, err)
			}
		}()
	}

	if err != nil {
		return fmt.Errorf(`Error while opening file: %v`, err)
	}

	if _, err = io.Copy(storageFile, content); err != nil {
		return fmt.Errorf(`Error while writing file: %v`, err)
	}

	return nil
}

// Rename file or directory from storage
func (a App) Rename(oldName, newName string) error {
	if err := a.checkPathname(oldName); err != nil {
		return err
	}

	if err := a.checkPathname(newName); err != nil {
		return err
	}

	_, err := a.Info(oldName)
	if err != nil {
		return fmt.Errorf(`Error while getting infos about %s: %v`, oldName, err)
	}

	_, err = a.Info(newName)
	if err == nil {
		return fmt.Errorf(`%s already exists`, newName)
	}

	if !provider.IsNotExist(err) {
		return fmt.Errorf(`Error while getting infos about %s: %v`, newName, err)
	}

	return os.Rename(a.getFullPath(oldName), a.getFullPath(newName))
}

// Remove file or directory from storage
func (a App) Remove(pathname string) error {
	if err := a.checkPathname(pathname); err != nil {
		return err
	}

	return os.RemoveAll(a.getFullPath(pathname))
}
