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
	"github.com/ViBiOh/fibr/pkg/sha"
	"github.com/ViBiOh/httputils/v3/pkg/flags"
	"github.com/ViBiOh/httputils/v3/pkg/httperror"
	"github.com/ViBiOh/httputils/v3/pkg/logger"
	"github.com/ViBiOh/httputils/v3/pkg/request"
)

const (
	// ThumbnailWidth is the width of each thumbnail generated
	ThumbnailWidth = 150

	// ThumbnailHeight is the width of each thumbnail generated
	ThumbnailHeight = 150
)

var (
	ignoredThumbnailDir = map[string]bool{
		"vendor":       true,
		"vendors":      true,
		"node_modules": true,
	}
)

// App of package
type App interface {
	Generate()
	HasThumbnail(*provider.StorageItem) (string, bool)
	Serve(http.ResponseWriter, *http.Request, *provider.StorageItem)
	List(http.ResponseWriter, *http.Request, *provider.StorageItem)
	AsyncGenerateThumbnail(*provider.StorageItem)
}

// Config of package
type Config struct {
	imaginaryURL *string
}

type app struct {
	thumbnailURL  string
	storage       provider.Storage
	pathnameInput chan *provider.StorageItem
}

// Flags adds flags for configuring package
func Flags(fs *flag.FlagSet, prefix string) Config {
	return Config{
		imaginaryURL: flags.New(prefix, "thumbnail").Name("ImaginaryURL").Default("http://image:9000").Label("Imaginary URL").ToString(fs),
	}
}

// New creates new App from Config
func New(config Config, storage provider.Storage) App {
	imaginaryURL := strings.TrimSpace(*config.imaginaryURL)
	if len(imaginaryURL) == 0 {
		return &app{}
	}

	app := &app{
		thumbnailURL:  fmt.Sprintf("%s/crop?width=%d&height=%d&stripmeta=true&noprofile=true&quality=80&type=jpeg", imaginaryURL, ThumbnailWidth, ThumbnailHeight),
		storage:       storage,
		pathnameInput: make(chan *provider.StorageItem, 10),
	}

	go app.Start()

	return app
}

// Enabled checks if app is enabled
func (a app) Enabled() bool {
	return len(a.thumbnailURL) != 0 && a.storage != nil
}

// HasThumbnail determine if thumbnail exist for given pathname
func (a app) HasThumbnail(item *provider.StorageItem) (string, bool) {
	if !a.Enabled() {
		return "", false
	}

	thumbnailPath := getThumbnailPath(item)

	_, err := a.storage.Info(thumbnailPath)
	return thumbnailPath, err == nil
}

func (a app) Start() {
	waitTimeout := time.Millisecond * 300

	for item := range a.pathnameInput {
		// Do not stress API
		time.Sleep(waitTimeout)

		if err := a.generateThumbnail(item); err != nil {
			logger.Error("%s", err)
		} else {
			logger.Info("Thumbnail generated for %s", item.Pathname)
		}
	}
}

// Serve check if thumbnail is present and serve it
func (a app) Serve(w http.ResponseWriter, r *http.Request, item *provider.StorageItem) {
	if CanHaveThumbnail(item) {
		if thumbnailPath, ok := a.HasThumbnail(item); ok {
			a.storage.Serve(w, r, thumbnailPath)
			return
		}
	}

	w.WriteHeader(http.StatusNoContent)
}

// List return all thumbnail in a base64 form
func (a app) List(w http.ResponseWriter, r *http.Request, item *provider.StorageItem) {
	if !a.Enabled() {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	items, err := a.storage.List(item.Pathname)
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

		thumbnailPath, ok := a.HasThumbnail(item)
		if !ok {
			continue
		}

		file, err := a.storage.ReaderFrom(thumbnailPath)
		if err != nil {
			logger.Error("unable to open %s: %s", item.Pathname, err)
		}

		content, err := ioutil.ReadAll(file)
		if err != nil {
			logger.Error("unable to read %s: %s", item.Pathname, err)
		}

		if commaNeeded {
			safeWrite(w, ",")
		} else {
			commaNeeded = true
		}

		safeWrite(w, fmt.Sprintf(`"%s":`, sha.Sha1(item.Name)))
		safeWrite(w, fmt.Sprintf(`"%s"`, base64.StdEncoding.EncodeToString(content)))
	}

	safeWrite(w, "}")
}

func (a app) generateThumbnail(item *provider.StorageItem) error {
	file, err := a.storage.ReaderFrom(item.Pathname)
	if err != nil {
		return err
	}

	ctx, cancel := getCtx(context.Background())
	defer cancel()

	resp, err := request.New().Post(a.thumbnailURL).Send(ctx, file)
	if err != nil {
		return err
	}

	thumbnailPath := getThumbnailPath(item)
	if err := a.storage.CreateDir(path.Dir(thumbnailPath)); err != nil {
		return err
	}

	if err := a.storage.Store(thumbnailPath, resp.Body); err != nil {
		return err
	}

	return nil
}

// Generate thumbnail for all storage
func (a app) Generate() {
	if !a.Enabled() {
		return
	}

	err := a.storage.Walk(func(item *provider.StorageItem, _ error) error {
		if item.IsDir && strings.HasPrefix(item.Name, ".") || ignoredThumbnailDir[item.Name] {
			return filepath.SkipDir
		}

		if !CanHaveThumbnail(item) {
			return nil
		}

		if _, ok := a.HasThumbnail(item); ok {
			return nil
		}

		a.AsyncGenerateThumbnail(item)

		return nil
	})

	if err != nil {
		logger.Error("%s", err)
	}
}

// AsyncGenerateThumbnail generate thumbnail image for given path
func (a app) AsyncGenerateThumbnail(item *provider.StorageItem) {
	if !a.Enabled() {
		return
	}

	a.pathnameInput <- item
}
