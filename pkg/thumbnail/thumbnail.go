package thumbnail

import (
	"bytes"
	"encoding/base64"
	"flag"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/fibr/pkg/sha"
	"github.com/ViBiOh/httputils/v4/pkg/flags"
	"github.com/ViBiOh/httputils/v4/pkg/httperror"
	"github.com/ViBiOh/httputils/v4/pkg/logger"
	"github.com/prometheus/client_golang/prometheus"
)

const (
	// Width is the width of each thumbnail generated
	Width = 150

	// Height is the width of each thumbnail generated
	Height = 150
)

// App of package
type App interface {
	Start()
	Delete(provider.StorageItem)
	Rename(provider.StorageItem, provider.StorageItem)

	HasThumbnail(provider.StorageItem) bool
	Serve(http.ResponseWriter, *http.Request, provider.StorageItem)
	List(http.ResponseWriter, *http.Request, provider.StorageItem)
	GenerateThumbnail(provider.StorageItem)
}

// Config of package
type Config struct {
	imageURL *string
	videoURL *string
}

type app struct {
	storageApp    provider.Storage
	prometheus    prometheus.Registerer
	pathnameInput chan provider.StorageItem

	imageURL string
	videoURL string
}

// Flags adds flags for configuring package
func Flags(fs *flag.FlagSet, prefix string) Config {
	return Config{
		imageURL: flags.New(prefix, "thumbnail").Name("ImageURL").Default("http://image:9000").Label("Imaginary URL").ToString(fs),
		videoURL: flags.New(prefix, "vith").Name("VideoURL").Default("http://video:1080").Label("Video Thumbnail URL").ToString(fs),
	}
}

// New creates new App from Config
func New(config Config, storage provider.Storage, prometheusApp prometheus.Registerer) App {
	imageURL := strings.TrimSpace(*config.imageURL)
	if len(imageURL) == 0 {
		return &app{}
	}

	videoURL := strings.TrimSpace(*config.videoURL)
	if len(videoURL) == 0 {
		return &app{}
	}

	app := &app{
		imageURL: fmt.Sprintf("%s/crop?width=%d&height=%d&stripmeta=true&noprofile=true&quality=80&type=jpeg", imageURL, Width, Height),
		videoURL: videoURL,

		storageApp: storage,
		prometheus: prometheusApp,

		pathnameInput: make(chan provider.StorageItem, 10),
	}

	return app
}

func (a app) enabled() bool {
	return len(a.imageURL) != 0 && len(a.videoURL) != 0
}

// Serve check if thumbnail is present and serve it
func (a app) Serve(w http.ResponseWriter, r *http.Request, item provider.StorageItem) {
	if !CanHaveThumbnail(item) || !a.HasThumbnail(item) {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	file, err := a.storageApp.ReaderFrom(getThumbnailPath(item))
	if file != nil {
		defer func() {
			if err := file.Close(); err != nil {
				logger.Error("unable to close thumbnail file: %s", err)
			}
		}()
	}
	if err != nil {
		httperror.InternalServerError(w, err)
		return
	}

	http.ServeContent(w, r, item.Name, item.Date, file)
}

// List return all thumbnail in a base64 form
func (a app) List(w http.ResponseWriter, _ *http.Request, item provider.StorageItem) {
	if !a.enabled() {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	items, err := a.storageApp.List(item.Pathname)
	if err != nil {
		httperror.InternalServerError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	w.WriteHeader(http.StatusOK)

	commaNeeded := false
	provider.SafeWrite(w, "{")

	for _, item := range items {
		if item.IsDir || !a.HasThumbnail(item) {
			continue
		}

		if commaNeeded {
			provider.SafeWrite(w, ",")
		} else {
			commaNeeded = true
		}

		provider.SafeWrite(w, fmt.Sprintf(`"%s":"`, sha.Sha1(item.Name)))
		a.encodeThumbnailContent(base64.NewEncoder(base64.StdEncoding, w), item)
		provider.SafeWrite(w, `"`)
	}

	provider.SafeWrite(w, "}")
}

func (a app) encodeThumbnailContent(encoder io.WriteCloser, item provider.StorageItem) {
	defer func() {
		if err := encoder.Close(); err != nil {
			logger.Error("unable to close encoder: %s", err)
		}
	}()

	file, err := a.storageApp.ReaderFrom(getThumbnailPath(item))
	if file != nil {
		defer func() {
			if err := file.Close(); err != nil {
				logger.Error("unable to close thumbnail item: %s", err)
			}
		}()
	}
	if err != nil {
		logger.Error("unable to open %s: %s", item.Pathname, err)
	}

	buffer := provider.BufferPool.Get().(*bytes.Buffer)
	defer provider.BufferPool.Put(buffer)

	if _, err := io.CopyBuffer(encoder, file, buffer.Bytes()); err != nil {
		logger.Error("unable to copy thumbnail: %s", err)
	}
}
