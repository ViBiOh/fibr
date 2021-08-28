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
type App struct {
	storageApp    provider.Storage
	pathnameInput chan provider.StorageItem

	counter *prometheus.CounterVec

	imageURL     string
	videoURL     string
	maxSize      int64
	minBitrate   uint64
	directAccess bool
}

// Config of package
type Config struct {
	imageURL     *string
	videoURL     *string
	maxSize      *int
	minBitrate   *uint
	directAccess *bool
}

// Flags adds flags for configuring package
func Flags(fs *flag.FlagSet, prefix string) Config {
	return Config{
		directAccess: flags.New(prefix, "vith", "DirectAccess").Default(false, nil).Label("Use Vith with direct access to filesystem (no large file upload to it, send a GET request)").ToBool(fs),
		imageURL:     flags.New(prefix, "thumbnail", "ImageURL").Default("http://image:9000", nil).Label("Imaginary URL").ToString(fs),
		maxSize:      flags.New(prefix, "thumbnail", "MaxSize").Default(1024*1024*200, nil).Label("Maximum file size (in bytes) for generating thumbnail (0 to no limit)").ToInt(fs),
		minBitrate:   flags.New(prefix, "vith", "MinBitrate").Default(80*1000*1000, nil).Label("Minimal video bitrate (in bits per second) to generate a streamable version (in HLS), if DirectAccess enabled").ToUint(fs),
		videoURL:     flags.New(prefix, "vith", "VideoURL").Default("http://video:1080", nil).Label("Video Thumbnail URL").ToString(fs),
	}
}

// New creates new App from Config
func New(config Config, storage provider.Storage, prometheusRegisterer prometheus.Registerer) (App, error) {
	imageURL := strings.TrimSpace(*config.imageURL)
	if len(imageURL) == 0 {
		return App{}, nil
	}

	videoURL := strings.TrimSpace(*config.videoURL)
	if len(videoURL) == 0 {
		return App{}, nil
	}

	counter, err := createMetric(prometheusRegisterer)
	if err != nil {
		return App{}, err
	}

	return App{
		imageURL:      fmt.Sprintf("%s/crop?width=%d&height=%d&stripmeta=true&noprofile=true&quality=80&type=jpeg", imageURL, Width, Height),
		videoURL:      videoURL,
		maxSize:       int64(*config.maxSize),
		minBitrate:    uint64(*config.minBitrate),
		directAccess:  *config.directAccess,
		storageApp:    storage,
		counter:       counter,
		pathnameInput: make(chan provider.StorageItem, 10),
	}, nil
}

func (a App) enabled() bool {
	return len(a.imageURL) != 0 && len(a.videoURL) != 0
}

// Stream check if stream is present and serve it
func (a App) Stream(w http.ResponseWriter, r *http.Request, item provider.StorageItem) {
	if !a.HasStream(item) {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	file, err := a.storageApp.ReaderFrom(getStreamPath(item))
	if err != nil {
		httperror.InternalServerError(w, err)
		return
	}

	defer func() {
		if err := file.Close(); err != nil {
			logger.Error("unable to close stream file: %s", err)
		}
	}()

	http.ServeContent(w, r, item.Name, item.Date, file)
}

// Serve check if thumbnail is present and serve it
func (a App) Serve(w http.ResponseWriter, r *http.Request, item provider.StorageItem) {
	if !a.CanHaveThumbnail(item) || !a.HasThumbnail(item) {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	file, err := a.storageApp.ReaderFrom(getThumbnailPath(item))
	if err != nil {
		httperror.InternalServerError(w, err)
		return
	}

	defer func() {
		if err := file.Close(); err != nil {
			logger.Error("unable to close thumbnail file: %s", err)
		}
	}()

	http.ServeContent(w, r, item.Name, item.Date, file)
}

// List return all thumbnail in a base64 form
func (a App) List(w http.ResponseWriter, _ *http.Request, item provider.StorageItem) {
	if !a.enabled() {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	items, err := a.storageApp.List(item.Pathname)
	if err != nil {
		httperror.InternalServerError(w, err)
		return
	}

	w.Header().Add("Content-Type", "application/json; charset=utf-8")
	w.Header().Add("Cache-Control", "no-cache")
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

func (a App) encodeThumbnailContent(encoder io.WriteCloser, item provider.StorageItem) {
	defer func() {
		if err := encoder.Close(); err != nil {
			logger.Error("unable to close encoder: %s", err)
		}
	}()

	file, err := a.storageApp.ReaderFrom(getThumbnailPath(item))
	if err != nil {
		logger.Error("unable to open %s: %s", item.Pathname, err)
	}

	defer func() {
		if err := file.Close(); err != nil {
			logger.Error("unable to close thumbnail item: %s", err)
		}
	}()

	buffer := provider.BufferPool.Get().(*bytes.Buffer)
	defer provider.BufferPool.Put(buffer)

	if _, err := io.CopyBuffer(encoder, file, buffer.Bytes()); err != nil {
		logger.Error("unable to copy thumbnail: %s", err)
	}
}
