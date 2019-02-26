package thumbnail

import (
	"context"
	"encoding/base64"
	"flag"
	"fmt"
	"io"
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
		`vendor`:       true,
		`vendors`:      true,
		`node_modules`: true,
	}
)

func getCtx(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx, defaultTimeout)
}

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
		pathnameInput: make(chan string, 10),
	}

	go func() {
		for pathname := range app.pathnameInput {
			// Do not stress API
			time.Sleep(waitTimeout)

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
	fullPath := path.Join(provider.MetadataDirectoryName, pathname)

	return fmt.Sprintf(`%s.png`, strings.TrimSuffix(fullPath, path.Ext(fullPath)))
}

// CanHaveThumbnail determine if thumbnail can be generated for given pathname
func (a App) CanHaveThumbnail(pathname string) bool {
	extension := strings.ToLower(path.Ext(pathname))

	return provider.ImageExtensions[extension] || provider.PdfExtensions[extension]
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

func safeWrite(w io.Writer, content string) {
	if _, err := io.WriteString(w, content); err != nil {
		logger.Error(`%+v`, errors.WithStack(err))
	}
}

// List return all thumbnail in a base64 form
func (a App) List(w http.ResponseWriter, r *http.Request, pathname string) {
	items, err := a.storage.List(pathname)
	if err != nil {
		httperror.InternalServerError(w, err)
		return
	}

	w.Header().Set(`Content-Type`, `application/json; charset=utf-8`)
	w.Header().Set(`Cache-Control`, `no-cache`)
	w.WriteHeader(http.StatusOK)

	safeWrite(w, `{`)

	for index, item := range items {
		if item.IsDir || !a.HasThumbnail(item.Pathname) {
			continue
		}

		file, err := a.storage.Read(getThumbnailPath(item.Pathname))
		if err != nil {
			logger.Error(`unable to open %s: %+v`, item.Pathname, err)
		}

		content, err := ioutil.ReadAll(file)
		if err != nil {
			logger.Error(`unable to read %s: %+v`, item.Pathname, errors.WithStack(err))
		}

		if index != 0 {
			safeWrite(w, `,`)
		}
		safeWrite(w, fmt.Sprintf(`"%s":`, tools.Sha1(item.Name)))
		safeWrite(w, fmt.Sprintf(`"%s"`, base64.StdEncoding.EncodeToString(content)))

	}

	safeWrite(w, `}`)
}

// Generate thumbnail for all storage
func (a App) Generate() {
	err := a.storage.Walk(func(item *provider.StorageItem, _ error) error {
		if item.IsDir && strings.HasPrefix(item.Name, `.`) || ignoredThumbnailDir[item.Name] {
			return filepath.SkipDir
		}

		if !a.CanHaveThumbnail(item.Pathname) || a.HasThumbnail(item.Pathname) {
			return nil
		}

		a.AsyncGenerateThumbnail(item.Pathname)

		return nil
	})

	if err != nil {
		logger.Error(`%+v`, err)
	}
}

// AsyncGenerateThumbnail generate thumbnail image for given path
func (a App) AsyncGenerateThumbnail(pathname string) {
	a.pathnameInput <- pathname
}

func (a App) generateThumbnail(pathname string) error {
	file, err := a.storage.Read(pathname)
	if err != nil {
		return err
	}

	ctx, cancel := getCtx(context.Background())
	defer cancel()

	result, _, _, err := request.Do(ctx, http.MethodPost, fmt.Sprintf(`%s/crop?width=150&height=150&stripmeta=true&type=png`, a.imaginaryURL), file, nil)
	if err != nil {
		return err
	}

	thumbnailPath := getThumbnailPath(pathname)
	if err := a.storage.Create(path.Dir(thumbnailPath)); err != nil {
		return err
	}

	if err := a.storage.Upload(thumbnailPath, result); err != nil {
		return err
	}

	return nil
}
