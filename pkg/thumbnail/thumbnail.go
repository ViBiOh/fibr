package thumbnail

import (
	"bytes"
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
	"github.com/ViBiOh/httputils/pkg/httpjson"
	"github.com/ViBiOh/httputils/pkg/logger"
	"github.com/ViBiOh/httputils/pkg/request"
	"github.com/ViBiOh/httputils/pkg/tools"
)

const (
	defaultTimeout = time.Second * 30
)

var (
	ignoredThumbnailDir = map[string]bool{
		`vendor`:       true,
		`vendors`:      true,
		`node_modules`: true,
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
		imaginaryURL: fs.String(tools.ToCamel(fmt.Sprintf(`%sImaginaryURL`, prefix)), `http://image:9000`, `[thumbnail] Imaginary URL`),
	}
}

// New creates new App from Config
func New(config Config, storage provider.Storage) *App {
	app := &App{
		imaginaryURL:  strings.TrimSpace(*config.imaginaryURL),
		storage:       storage,
		pathnameInput: make(chan string),
	}

	go func() {
		for pathname := range app.pathnameInput {
			if err := app.generateThumbnail(pathname); err != nil {
				logger.Error(`%+v`, err)
			} else {
				logger.Info(`Thumbnail generated for %s`, pathname)
			}
		}
	}()

	return app
}

func getThumbnailPath(pathname string) string {
	return path.Join(provider.MetadataDirectoryName, pathname)
}

// HasThumbnail determine if thumbnail exist for given pathname
func (a App) HasThumbnail(pathname string) bool {
	_, err := a.storage.Info(getThumbnailPath(pathname))
	return err == nil
}

// ServeIfPresent check if thumbnail is present and serve it
func (a App) ServeIfPresent(w http.ResponseWriter, r *http.Request, pathname string) bool {
	if a.HasThumbnail(pathname) {
		a.storage.Serve(w, r, getThumbnailPath(pathname))
		return true
	}

	return false
}

// List return all thumbnail in a base64 form
func (a App) List(w http.ResponseWriter, r *http.Request, pathname string) {
	items, err := a.storage.List(pathname)
	if err != nil {
		httperror.InternalServerError(w, err)
		return
	}

	thumbnails := make(map[string]string)

	for _, item := range items {
		if item.IsDir || !a.HasThumbnail(item.Pathname) {
			continue
		}

		file, err := a.storage.Read(getThumbnailPath(item.Pathname))
		if err != nil {
			httperror.InternalServerError(w, err)
			return
		}

		content, err := ioutil.ReadAll(file)
		if err != nil {
			httperror.InternalServerError(w, errors.WithStack(err))
			return
		}

		thumbnails[tools.Sha1(item.Name)] = base64.StdEncoding.EncodeToString(content)
	}

	if err := httpjson.ResponseJSON(w, http.StatusOK, thumbnails, httpjson.IsPretty(r)); err != nil {
		httperror.InternalServerError(w, err)
	}
}

// Generate thumbnail for all storage
func (a App) Generate() {
	err := a.storage.Walk(func(pathname string, item *provider.StorageItem, _ error) error {
		if item.IsDir && strings.HasPrefix(item.Name, `.`) || ignoredThumbnailDir[item.Name] {
			return filepath.SkipDir
		}

		if !provider.ImageExtensions[item.Extension()] || a.HasThumbnail(pathname) {
			return nil
		}

		a.AsyncGenerateThumbnail(pathname)

		return nil

	})

	if err != nil {
		logger.Error(`%+v`, err)
	}
}

func getCtx(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx, defaultTimeout)
}

// AsyncGenerateThumbnail generate thumbnail image for given path
func (a App) AsyncGenerateThumbnail(pathname string) {
	a.pathnameInput <- pathname
}

func (a App) generateThumbnail(pathname string) error {
	file, err := a.storage.Read(pathname)
	if file != nil {
		defer func() {
			if err := file.Close(); err != nil {
				logger.Error(`%+v`, err)
			}
		}()
	}
	if err != nil {
		return err
	}

	payload, err := ioutil.ReadAll(file)
	if err != nil {
		return errors.WithStack(err)
	}

	ctx, cancel := getCtx(context.Background())
	defer cancel()

	headers := http.Header{}
	headers.Set(`Content-Type`, `image/*`)
	headers.Set(`Accept`, `image/*`)
	result, _, _, err := request.Do(ctx, http.MethodPost, fmt.Sprintf(`%s/smartcrop?width=150&height=150&stripmeta=true`, a.imaginaryURL), payload, headers)
	if err != nil {
		return err
	}

	thumbnailPath := getThumbnailPath(pathname)
	if err := a.storage.Create(path.Dir(thumbnailPath)); err != nil {
		return err
	}

	if err := a.storage.Upload(thumbnailPath, ioutil.NopCloser(bytes.NewReader(result))); err != nil {
		return err
	}

	return nil
}
