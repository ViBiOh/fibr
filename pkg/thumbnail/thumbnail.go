package thumbnail

import (
	"encoding/base64"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/fibr/pkg/sha"
	"github.com/ViBiOh/httputils/v3/pkg/flags"
	"github.com/ViBiOh/httputils/v3/pkg/httperror"
	"github.com/ViBiOh/httputils/v3/pkg/logger"
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
	Remove(provider.StorageItem)
	Rename(provider.StorageItem, provider.StorageItem)
	HasThumbnail(provider.StorageItem) bool
	Serve(http.ResponseWriter, *http.Request, provider.StorageItem)
	List(http.ResponseWriter, *http.Request, provider.StorageItem)
	AsyncGenerateThumbnail(provider.StorageItem)
}

// Config of package
type Config struct {
	imageURL *string
	videoURL *string
}

type app struct {
	imageURL      string
	videoURL      string
	storage       provider.Storage
	pathnameInput chan provider.StorageItem
}

// Flags adds flags for configuring package
func Flags(fs *flag.FlagSet, prefix string) Config {
	return Config{
		imageURL: flags.New(prefix, "thumbnail").Name("imageURL").Default("http://image:9000").Label("Imaginary URL").ToString(fs),
		videoURL: flags.New(prefix, "vith").Name("VideoURL").Default("http://video:1080").Label("Video Thumbnail URL").ToString(fs),
	}
}

// New creates new App from Config
func New(config Config, storage provider.Storage) App {
	imageURL := strings.TrimSpace(*config.imageURL)
	if len(imageURL) == 0 {
		return &app{}
	}

	videoURL := strings.TrimSpace(*config.videoURL)
	if len(videoURL) == 0 {
		return &app{}
	}

	app := &app{
		imageURL:      fmt.Sprintf("%s/crop?width=%d&height=%d&stripmeta=true&noprofile=true&quality=80&type=jpeg", imageURL, ThumbnailWidth, ThumbnailHeight),
		videoURL:      videoURL,
		storage:       storage,
		pathnameInput: make(chan provider.StorageItem, 10),
	}

	go app.Start()

	return app
}

// Enabled checks if app is enabled
func (a app) Enabled() bool {
	return len(a.imageURL) != 0 && len(a.videoURL) != 0 && a.storage != nil
}

// Serve check if thumbnail is present and serve it
func (a app) Serve(w http.ResponseWriter, r *http.Request, item provider.StorageItem) {
	if !CanHaveThumbnail(item) || !a.HasThumbnail(item) {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	file, err := a.storage.ReaderFrom(getThumbnailPath(item))
	if err != nil {
		httperror.InternalServerError(w, err)
		return
	}

	http.ServeContent(w, r, item.Name, item.Date, file)
}

// List return all thumbnail in a base64 form
func (a app) List(w http.ResponseWriter, r *http.Request, item provider.StorageItem) {
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
	provider.SafeWrite(w, "{")

	for _, item := range items {
		if item.IsDir || !a.HasThumbnail(item) {
			continue
		}

		file, err := a.storage.ReaderFrom(getThumbnailPath(item))
		if err != nil {
			logger.Error("unable to open %s: %s", item.Pathname, err)
		}

		content, err := ioutil.ReadAll(file)
		if err != nil {
			logger.Error("unable to read %s: %s", item.Pathname, err)
		}

		if commaNeeded {
			provider.SafeWrite(w, ",")
		} else {
			commaNeeded = true
		}

		provider.SafeWrite(w, fmt.Sprintf(`"%s":`, sha.Sha1(item.Name)))
		provider.SafeWrite(w, fmt.Sprintf(`"%s"`, base64.StdEncoding.EncodeToString(content)))
	}

	provider.SafeWrite(w, "}")
}
