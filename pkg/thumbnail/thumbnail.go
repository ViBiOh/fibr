package thumbnail

import (
	"bytes"
	"encoding/base64"
	"flag"
	"fmt"
	"io"
	"net/http"
	"path"
	"strings"
	"time"

	absto "github.com/ViBiOh/absto/pkg/model"
	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/flags"
	"github.com/ViBiOh/httputils/v4/pkg/amqp"
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

var cacheDuration string = fmt.Sprintf("private, max-age=%.0f", time.Duration(time.Minute*5).Seconds())

// App of package
type App struct {
	storageApp    absto.Storage
	pathnameInput chan absto.Item
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
		vithURL:  flags.New(prefix, "thumbnail", "URL").Default("http://vith:1080", nil).Label("Vith Thumbnail URL").ToString(fs),
		vithUser: flags.New(prefix, "thumbnail", "User").Default("", nil).Label("Vith Thumbnail Basic Auth User").ToString(fs),
		vithPass: flags.New(prefix, "thumbnail", "Password").Default("", nil).Label("Vith Thumbnail Basic Auth Password").ToString(fs),

		directAccess: flags.New(prefix, "thumbnail", "DirectAccess").Default(false, nil).Label("Use Vith with direct access to filesystem (no large file upload, send a GET request, Basic Auth recommended)").ToBool(fs),
		maxSize:      flags.New(prefix, "thumbnail", "MaxSize").Default(1024*1024*200, nil).Label("Maximum file size (in bytes) for generating thumbnail (0 to no limit). Not used if DirectAccess enabled.").ToInt64(fs),
		minBitrate:   flags.New(prefix, "thumbnail", "MinBitrate").Default(80*1000*1000, nil).Label("Minimal video bitrate (in bits per second) to generate a streamable version (in HLS), if DirectAccess enabled").ToUint64(fs),

		amqpExchange:            flags.New(prefix, "thumbnail", "AmqpExchange").Default("fibr", nil).Label("AMQP Exchange Name").ToString(fs),
		amqpStreamRoutingKey:    flags.New(prefix, "thumbnail", "AmqpStreamRoutingKey").Default("stream", nil).Label("AMQP Routing Key for stream").ToString(fs),
		amqpThumbnailRoutingKey: flags.New(prefix, "thumbnail", "AmqpThumbnailRoutingKey").Default("thumbnail", nil).Label("AMQP Routing Key for thumbnail").ToString(fs),
	}
}

// New creates new App from Config
func New(config Config, storage absto.Storage, prometheusRegisterer prometheus.Registerer, amqpClient *amqp.Client) (App, error) {
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
		pathnameInput: make(chan absto.Item, 10),
	}, nil
}

// Stream check if stream is present and serve it
func (a App) Stream(w http.ResponseWriter, r *http.Request, item absto.Item) {
	if !a.HasStream(item) {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	reader, err := a.storageApp.ReadFrom(getStreamPath(item))
	if err != nil {
		httperror.InternalServerError(w, err)
		return
	}

	defer func() {
		if closeErr := reader.Close(); closeErr != nil {
			logger.WithField("fn", "thumbnail.Stream").WithField("item", item.Pathname).Error("unable to close: %s", closeErr)
		}
	}()

	w.Header().Add("Content-Type", "application/x-mpegURL")
	http.ServeContent(w, r, item.Name, item.Date, reader)
}

// Serve check if thumbnail is present and serve it
func (a App) Serve(w http.ResponseWriter, r *http.Request, item absto.Item) {
	if !a.CanHaveThumbnail(item) || !a.HasThumbnail(item) {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	thumbnailPath := getThumbnailPath(item)
	reader, err := a.storageApp.ReadFrom(thumbnailPath)
	if err != nil {
		httperror.InternalServerError(w, err)
		return
	}

	defer func() {
		if closeErr := reader.Close(); closeErr != nil {
			logger.WithField("fn", "thumbnail.Serve").WithField("item", item.Pathname).WithField("item", item.Pathname).Error("unable to close: %s", closeErr)
		}
	}()

	w.Header().Add("Cache-Control", cacheDuration)
	w.Header().Add("Content-Disposition", fmt.Sprintf("inline; filename=%s", path.Base(thumbnailPath)))

	http.ServeContent(w, r, path.Base(thumbnailPath), item.Date, reader)
}

// List return all thumbnail in a base64 form
func (a App) List(w http.ResponseWriter, r *http.Request, items []absto.Item) {
	if len(items) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	etag := fmt.Sprintf(`W/"%s"`, a.thumbnailHash(items))

	if r.Header.Get("If-None-Match") == etag {
		w.WriteHeader(http.StatusNotModified)
		return
	}

	w.Header().Add("Content-Type", "text/plain; charset=utf-8")
	w.Header().Add("Etag", etag)
	w.WriteHeader(http.StatusOK)

	done := r.Context().Done()
	isDone := func() bool {
		select {
		case <-done:
			return true
		default:
			return false
		}
	}

	for _, item := range items {
		if isDone() {
			return
		}

		if !a.HasThumbnail(item) {
			continue
		}

		provider.DoneWriter(isDone, w, sha.New(item.Name))
		provider.DoneWriter(isDone, w, `,`)
		a.encodeContent(base64.NewEncoder(base64.StdEncoding, w), item)
		provider.DoneWriter(isDone, w, "\n")
	}
}

func (a App) thumbnailHash(items []absto.Item) string {
	hasher := Stream()

	for _, item := range items {
		if info, err := a.storageApp.Info(getThumbnailPath(item)); err == nil {
			hasher.Write(info)
		}
	}

	return hasher.Sum()
}

func (a App) encodeContent(encoder io.WriteCloser, item absto.Item) {
	reader, err := a.storageApp.ReadFrom(getThumbnailPath(item))
	if err != nil {
		logEncodeContentError(item).Error("unable to open: %s", err)
		return
	}

	buffer := provider.BufferPool.Get().(*bytes.Buffer)
	defer provider.BufferPool.Put(buffer)

	if _, err = io.CopyBuffer(encoder, reader, buffer.Bytes()); err != nil {
		logEncodeContentError(item).Error("unable to copy: %s", err)
	}

	if err = reader.Close(); err != nil {
		logEncodeContentError(item).Error("unable to close item: %s", err)
	}

	if err = encoder.Close(); err != nil {
		logEncodeContentError(item).Error("unable to close encoder: %s", err)
	}
}

func logEncodeContentError(item absto.Item) logger.Provider {
	return logger.WithField("fn", "thumbnail.encodeContent").WithField("item", item.Pathname)
}
