package thumbnail

import (
	"context"
	"encoding/base64"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/httputils/pkg/errors"
	"github.com/ViBiOh/httputils/pkg/httperror"
	"github.com/ViBiOh/httputils/pkg/logger"
	"github.com/ViBiOh/httputils/pkg/request"
	"github.com/ViBiOh/httputils/pkg/tools"
)

const (
	defaultTimeout = time.Second * 30
	waitTimeout    = time.Millisecond * 300
)

var (
	ignoredThumbnailDir = map[string]bool{
		"vendor":       true,
		"vendors":      true,
		"node_modules": true,
	}
)

// Config of package
type Config struct {
	imaginaryURL *string
}

// App of package
type App struct {
	imaginaryURL  string
	storage       provider.Storage
	pathnameInput chan string
}

// Flags adds flags for configuring package
func Flags(fs *flag.FlagSet, prefix string) Config {
	return Config{
		imaginaryURL: fs.String(tools.ToCamel(fmt.Sprintf("%sImaginaryURL", prefix)), "http://image:9000", "[thumbnail] Imaginary URL"),
	}
}

// New creates new App from Config
func New(config Config, storage provider.Storage) *App {
	if *config.imaginaryURL == "" {
		return &App{}
	}

	app := &App{
		imaginaryURL:  fmt.Sprintf("%s/crop?width=150&height=150&stripmeta=true&noprofile=true&quality=80&type=jpeg", *config.imaginaryURL),
		storage:       storage,
		pathnameInput: make(chan string, 10),
	}

	go func() {
		for pathname := range app.pathnameInput {
			// Do not stress API
			time.Sleep(waitTimeout)

			if err := app.generateThumbnail(pathname); err != nil {
				logger.Error("%#v", err)
			} else {
				logger.Info("Thumbnail generated for %s", pathname)
			}
		}
	}()

	return app
}

// Enabled checks if app is enabled
func (a App) Enabled() bool {
	return a.imaginaryURL != "" && a.storage != nil
}

// HasThumbnail determine if thumbnail exist for given pathname
func (a App) HasThumbnail(pathname string) (string, bool) {
	if !a.Enabled() {
		return "", false
	}

	thumbnailPath := getThumbnailPath(pathname)

	_, err := a.storage.Info(thumbnailPath)
	return thumbnailPath, err == nil
}

// ServeIfPresent check if thumbnail is present and serve it
func (a App) ServeIfPresent(w http.ResponseWriter, r *http.Request, pathname string) bool {
	if thumbnailPath, ok := a.HasThumbnail(pathname); ok {
		a.storage.Serve(w, r, thumbnailPath)
		return true
	}

	return false
}

// List return all thumbnail in a base64 form
func (a App) List(w http.ResponseWriter, r *http.Request, pathname string) {
	if !a.Enabled() {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	items, err := a.storage.List(pathname)
	if err != nil {
		httperror.InternalServerError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	w.WriteHeader(http.StatusOK)

	commaNeeded := false
	safeWrite(w, "{")

	for _, item := range items {
		if item.IsDir {
			continue
		}

		thumbnailPath, ok := a.HasThumbnail(item.Pathname)
		if !ok {
			continue
		}

		file, err := a.storage.ReaderFrom(thumbnailPath)
		if err != nil {
			logger.Error("unable to open %s: %#v", item.Pathname, err)
		}

		content, err := ioutil.ReadAll(file)
		if err != nil {
			logger.Error("unable to read %s: %#v", item.Pathname, errors.WithStack(err))
		}

		if commaNeeded {
			safeWrite(w, ",")
		} else {
			commaNeeded = true
		}

		safeWrite(w, fmt.Sprintf(`"%s":`, tools.Sha1(item.Name)))
		safeWrite(w, fmt.Sprintf(`"%s"`, base64.StdEncoding.EncodeToString(content)))
	}

	safeWrite(w, "}")
}

func (a App) generateThumbnail(pathname string) error {
	file, err := a.storage.ReaderFrom(pathname)
	if err != nil {
		return err
	}

	ctx, cancel := getCtx(context.Background())
	defer cancel()

	result, _, _, err := request.Do(ctx, http.MethodPost, a.imaginaryURL, file, nil)
	if err != nil {
		return err
	}

	thumbnailPath := getThumbnailPath(pathname)
	if err := a.storage.CreateDir(path.Dir(thumbnailPath)); err != nil {
		return err
	}

	if err := a.storage.Store(thumbnailPath, result); err != nil {
		return err
	}

	return nil
}

// Generate thumbnail for all storage
func (a App) Generate() {
	if !a.Enabled() {
		return
	}

	err := a.storage.Walk(func(item *provider.StorageItem, _ error) error {
		if item.IsDir && strings.HasPrefix(item.Name, ".") || ignoredThumbnailDir[item.Name] {
			return filepath.SkipDir
		}

		if !CanHaveThumbnail(item.Pathname) {
			return nil
		}

		if _, ok := a.HasThumbnail(item.Pathname); ok {
			return nil
		}

		a.AsyncGenerateThumbnail(item.Pathname)

		return nil
	})

	if err != nil {
		logger.Error("%#v", err)
	}
}

// AsyncGenerateThumbnail generate thumbnail image for given path
func (a App) AsyncGenerateThumbnail(pathname string) {
	if !a.Enabled() {
		return
	}

	a.pathnameInput <- pathname
}
