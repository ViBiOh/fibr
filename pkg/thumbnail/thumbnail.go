package thumbnail

import (
	"bytes"
	"context"
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
	// SmallSize is the square size of each thumbnail generated
	SmallSize uint64 = 150
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

	sizes     []uint64
	largeSize uint64

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

	largeSize *uint64
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

		largeSize: flags.New(prefix, "thumbnail", "LargeSize").Default(800, nil).Label("Size of large thumbnail for story display (thumbnail are always squared). 0 to disable").ToUint64(fs),
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

	largeSize := *config.largeSize
	var sizes []uint64
	if largeSize > 0 {
		sizes = []uint64{SmallSize, largeSize}
	} else {
		sizes = []uint64{SmallSize}
	}

	return App{
		vithRequest: request.New().URL(*config.vithURL).BasicAuth(*config.vithUser, *config.vithPass).WithClient(provider.SlowClient),

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

		largeSize: largeSize,
		sizes:     sizes,
	}, nil
}

// LargeThumbnailSize give large thumbnail size
func (a App) LargeThumbnailSize() uint64 {
	return a.largeSize
}

// Stream check if stream is present and serve it
func (a App) Stream(w http.ResponseWriter, r *http.Request, item absto.Item) {
	reader, err := a.storageApp.ReadFrom(r.Context(), getStreamPath(item))
	if err != nil {
		if absto.IsNotExist(err) {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		httperror.InternalServerError(w, err)
		return
	}

	defer provider.LogClose(reader, "thumbnail.Stream", item.Pathname)

	w.Header().Add("Content-Type", "application/x-mpegURL")
	http.ServeContent(w, r, item.Name, item.Date, reader)
}

// Serve check if thumbnail is present and serve it
func (a App) Serve(w http.ResponseWriter, r *http.Request, item absto.Item) {
	if !a.CanHaveThumbnail(item) {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	scale := SmallSize
	if rawScale := r.URL.Query().Get("scale"); len(rawScale) > 0 {
		if rawScale == "large" && a.largeSize > 0 {
			scale = a.largeSize
		}
	}

	ctx := r.Context()

	thumbnailInfo, ok := a.ThumbnailInfo(ctx, item, scale)
	if !ok {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	reader, err := a.storageApp.ReadFrom(ctx, thumbnailInfo.Pathname)
	if err != nil {
		httperror.InternalServerError(w, err)
		return
	}

	defer provider.LogClose(reader, "thumbnail.Serve", item.Pathname)

	w.Header().Add("Cache-Control", cacheDuration)
	w.Header().Add("Content-Disposition", fmt.Sprintf("inline; filename=%s", path.Base(thumbnailInfo.Pathname)))

	http.ServeContent(w, r, path.Base(thumbnailInfo.Pathname), item.Date, reader)
}

// List return all thumbnail in a base64 form
func (a App) List(w http.ResponseWriter, r *http.Request, items []absto.Item) {
	if len(items) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	ctx := r.Context()

	etag, ok := provider.EtagMatch(w, r, a.thumbnailHash(ctx, items))
	if ok {
		return
	}

	w.Header().Add("Content-Type", "text/plain; charset=utf-8")
	w.Header().Add("Etag", etag)
	w.WriteHeader(http.StatusOK)

	done := ctx.Done()
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

		thumbnailInfo, ok := a.ThumbnailInfo(ctx, item, SmallSize)
		if !ok {
			continue
		}

		provider.DoneWriter(isDone, w, item.ID)
		provider.DoneWriter(isDone, w, `,`)
		a.encodeContent(ctx, base64.NewEncoder(base64.StdEncoding, w), thumbnailInfo)
		provider.DoneWriter(isDone, w, "\n")
	}
}

func (a App) thumbnailHash(ctx context.Context, items []absto.Item) string {
	hasher := sha.Stream()

	for _, item := range items {
		if info, err := a.storageApp.Info(ctx, a.PathForScale(item, SmallSize)); err == nil {
			hasher.Write(info)
		}
	}

	return hasher.Sum()
}

func (a App) encodeContent(ctx context.Context, encoder io.WriteCloser, item absto.Item) {
	defer provider.LogClose(encoder, "thumbnail.encodeContent", "encoder")

	reader, err := a.storageApp.ReadFrom(ctx, item.Pathname)
	if err != nil {
		logEncodeContentError(item).Error("unable to open: %s", err)
		return
	}

	defer provider.LogClose(reader, "thumbnail.encodeContent", item.Pathname)

	buffer := provider.BufferPool.Get().(*bytes.Buffer)
	defer provider.BufferPool.Put(buffer)

	if _, err = io.CopyBuffer(encoder, reader, buffer.Bytes()); err != nil {
		logEncodeContentError(item).Error("unable to copy: %s", err)
	}
}

func logEncodeContentError(item absto.Item) logger.Provider {
	return logger.WithField("fn", "thumbnail.encodeContent").WithField("item", item.Pathname)
}
