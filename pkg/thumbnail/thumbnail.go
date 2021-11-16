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
	"github.com/ViBiOh/httputils/v4/pkg/amqp"
	"github.com/ViBiOh/httputils/v4/pkg/flags"
	"github.com/ViBiOh/httputils/v4/pkg/httperror"
	"github.com/ViBiOh/httputils/v4/pkg/logger"
	prom "github.com/ViBiOh/httputils/v4/pkg/prometheus"
	"github.com/ViBiOh/httputils/v4/pkg/request"
	"github.com/ViBiOh/httputils/v4/pkg/sha"
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
	metric        *prometheus.CounterVec

	amqpClient              *amqp.Client
	amqpExchange            string
	amqpStreamRoutingKey    string
	amqpThumbnailRoutingKey string

	vithRequest request.Request

	maxSize      int64
	minBitrate   uint64
	directAccess bool
}

// Config of package
type Config struct {
	vithURL  *string
	vithUser *string
	vithPass *string

	amqpExchange            *string
	amqpStreamRoutingKey    *string
	amqpThumbnailRoutingKey *string

	maxSize      *int64
	minBitrate   *uint64
	directAccess *bool
}

// Flags adds flags for configuring package
func Flags(fs *flag.FlagSet, prefix string) Config {
	return Config{
		vithURL:  flags.New(prefix, "vith", "VithURL").Default("http://vith:1080", nil).Label("Vith Thumbnail URL").ToString(fs),
		vithUser: flags.New(prefix, "vith", "VithUser").Default("", nil).Label("Vith Thumbnail Basic Auth User").ToString(fs),
		vithPass: flags.New(prefix, "vith", "VithPassword").Default("", nil).Label("Vith Thumbnail Basic Auth Password").ToString(fs),

		directAccess: flags.New(prefix, "vith", "DirectAccess").Default(false, nil).Label("Use Vith with direct access to filesystem (no large file upload, send a GET request, Basic Auth recommended)").ToBool(fs),
		maxSize:      flags.New(prefix, "thumbnail", "MaxSize").Default(1024*1024*200, nil).Label("Maximum file size (in bytes) for generating thumbnail (0 to no limit). Not used if DirectAccess enabled.").ToInt64(fs),
		minBitrate:   flags.New(prefix, "vith", "MinBitrate").Default(80*1000*1000, nil).Label("Minimal video bitrate (in bits per second) to generate a streamable version (in HLS), if DirectAccess enabled").ToUint64(fs),

		amqpExchange:            flags.New(prefix, "vith", "AmqpExchange").Default("fibr", nil).Label("AMQP Exchange Name").ToString(fs),
		amqpStreamRoutingKey:    flags.New(prefix, "vith", "AmqpStreamRoutingKey").Default("stream", nil).Label("AMQP Routing Key for stream").ToString(fs),
		amqpThumbnailRoutingKey: flags.New(prefix, "vith", "AmqpThumbnailRoutingKey").Default("thumbnail", nil).Label("AMQP Routing Key for thumbnail").ToString(fs),
	}
}

// New creates new App from Config
func New(config Config, storage provider.Storage, prometheusRegisterer prometheus.Registerer, amqpClient *amqp.Client) (App, error) {
	var amqpExchange string
	if amqpClient != nil {
		amqpExchange = strings.TrimSpace(*config.amqpExchange)

		if err := amqpClient.Publisher(amqpExchange, "direct", nil); err != nil {
			return App{}, fmt.Errorf("unable to configure amqp: %s", err)
		}
	}

	return App{
		vithRequest: request.New().URL(*config.vithURL).BasicAuth(*config.vithUser, *config.vithPass),

		maxSize:      *config.maxSize,
		minBitrate:   *config.minBitrate,
		directAccess: *config.directAccess,

		amqpExchange:            amqpExchange,
		amqpStreamRoutingKey:    strings.TrimSpace(*config.amqpStreamRoutingKey),
		amqpThumbnailRoutingKey: strings.TrimSpace(*config.amqpThumbnailRoutingKey),

		storageApp:    storage,
		amqpClient:    amqpClient,
		metric:        prom.CounterVec(prometheusRegisterer, "fibr", "thumbnail", "item", "type", "state"),
		pathnameInput: make(chan provider.StorageItem, 10),
	}, nil
}

func (a App) vithEnabled() bool {
	return !a.vithRequest.IsZero()
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

		provider.SafeWrite(w, `"`)
		provider.SafeWrite(w, sha.New(item.Name))
		provider.SafeWrite(w, `":"`)
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
