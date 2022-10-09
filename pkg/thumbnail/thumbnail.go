package thumbnail

import (
	"bytes"
	"context"
	"encoding/base64"
	"flag"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	absto "github.com/ViBiOh/absto/pkg/model"
	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/flags"
	"github.com/ViBiOh/httputils/v4/pkg/amqp"
	"github.com/ViBiOh/httputils/v4/pkg/cache"
	"github.com/ViBiOh/httputils/v4/pkg/httperror"
	"github.com/ViBiOh/httputils/v4/pkg/logger"
	prom "github.com/ViBiOh/httputils/v4/pkg/prometheus"
	"github.com/ViBiOh/httputils/v4/pkg/query"
	"github.com/ViBiOh/httputils/v4/pkg/redis"
	"github.com/ViBiOh/httputils/v4/pkg/request"
	"github.com/ViBiOh/httputils/v4/pkg/sha"
	"github.com/ViBiOh/httputils/v4/pkg/tracer"
	"github.com/prometheus/client_golang/prometheus"
	"go.opentelemetry.io/otel/trace"
)

const (
	// SmallSize is the square size of each thumbnail generated
	SmallSize uint64 = 150
)

var cacheDuration = fmt.Sprintf("private, max-age=%.0f", (time.Minute * 5).Seconds())

type App struct {
	redisClient     redis.App
	tracer          trace.Tracer
	storageApp      absto.Storage
	smallStorageApp absto.Storage
	largeStorageApp absto.Storage
	pathnameInput   chan absto.Item
	metric          *prometheus.CounterVec

	cacheApp cache.App[string, absto.Item]

	amqpClient              *amqp.Client
	amqpThumbnailRoutingKey string
	amqpExchange            string
	amqpStreamRoutingKey    string

	sizes        []uint64
	vithRequest  request.Request
	largeSize    uint64
	maxSize      int64
	minBitrate   uint64
	directAccess bool
}

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

func Flags(fs *flag.FlagSet, prefix string) Config {
	return Config{
		vithURL:  flags.String(fs, prefix, "thumbnail", "URL", "Vith Thumbnail URL", "http://vith:1080", nil),
		vithUser: flags.String(fs, prefix, "thumbnail", "User", "Vith Thumbnail Basic Auth User", "", nil),
		vithPass: flags.String(fs, prefix, "thumbnail", "Password", "Vith Thumbnail Basic Auth Password", "", nil),

		directAccess: flags.Bool(fs, prefix, "thumbnail", "DirectAccess", "Use Vith with direct access to filesystem (no large file upload, send a GET request, Basic Auth recommended)", false, nil),
		maxSize:      flags.Int64(fs, prefix, "thumbnail", "MaxSize", "Maximum file size (in bytes) for generating thumbnail (0 to no limit). Not used if DirectAccess enabled.", 1024*1024*200, nil),
		minBitrate:   flags.Uint64(fs, prefix, "thumbnail", "MinBitrate", "Minimal video bitrate (in bits per second) to generate a streamable version (in HLS), if DirectAccess enabled", 80*1000*1000, nil),

		amqpExchange:            flags.String(fs, prefix, "thumbnail", "AmqpExchange", "AMQP Exchange Name", "fibr", nil),
		amqpStreamRoutingKey:    flags.String(fs, prefix, "thumbnail", "AmqpStreamRoutingKey", "AMQP Routing Key for stream", "stream", nil),
		amqpThumbnailRoutingKey: flags.String(fs, prefix, "thumbnail", "AmqpThumbnailRoutingKey", "AMQP Routing Key for thumbnail", "thumbnail", nil),

		largeSize: flags.Uint64(fs, prefix, "thumbnail", "LargeSize", "Size of large thumbnail for story display (thumbnail are always squared). 0 to disable", 800, nil),
	}
}

func New(config Config, storage absto.Storage, redisClient redis.App, prometheusRegisterer prometheus.Registerer, tracerApp tracer.App, amqpClient *amqp.Client) (App, error) {
	var amqpExchange string
	if amqpClient != nil {
		amqpExchange = strings.TrimSpace(*config.amqpExchange)

		if err := amqpClient.Publisher(amqpExchange, "direct", nil); err != nil {
			return App{}, fmt.Errorf("configure amqp: %w", err)
		}
	}

	largeSize := *config.largeSize
	var sizes []uint64
	if largeSize > 0 {
		sizes = []uint64{SmallSize, largeSize}
	} else {
		sizes = []uint64{SmallSize}
	}

	app := App{
		vithRequest: request.New().URL(*config.vithURL).BasicAuth(*config.vithUser, *config.vithPass).WithClient(provider.SlowClient),

		maxSize:      *config.maxSize,
		minBitrate:   *config.minBitrate,
		directAccess: *config.directAccess,

		redisClient: redisClient,
		tracer:      tracerApp.GetTracer("thumbnail"),

		amqpExchange:            amqpExchange,
		amqpStreamRoutingKey:    strings.TrimSpace(*config.amqpStreamRoutingKey),
		amqpThumbnailRoutingKey: strings.TrimSpace(*config.amqpThumbnailRoutingKey),

		storageApp: storage,
		smallStorageApp: storage.WithIgnoreFn(func(item absto.Item) bool {
			return !strings.HasSuffix(item.Name, ".webp") || strings.HasSuffix(item.Name, "_large.webp")
		}),
		largeStorageApp: storage.WithIgnoreFn(func(item absto.Item) bool {
			return !strings.HasSuffix(item.Name, "_large.webp")
		}),
		amqpClient:    amqpClient,
		metric:        prom.CounterVec(prometheusRegisterer, "fibr", "thumbnail", "item", "type", "state"),
		pathnameInput: make(chan absto.Item, provider.MaxConcurrency),

		largeSize: largeSize,
		sizes:     sizes,
	}

	app.cacheApp = cache.New(redisClient, redisKey, func(ctx context.Context, pathname string) (absto.Item, error) {
		return app.storageApp.Info(ctx, pathname)
	}, redisCacheDuration, provider.MaxConcurrency, tracerApp.GetTracer("thumbnail_cache"))

	return app, nil
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

	name := a.PathForScale(item, scale)

	reader, err := a.storageApp.ReadFrom(ctx, name)
	if err != nil {
		if absto.IsNotExist(err) {
			w.WriteHeader(http.StatusNoContent)
		}

		httperror.InternalServerError(w, err)

		return
	}

	baseName := name

	defer provider.LogClose(reader, "thumbnail.Serve", item.Pathname)

	w.Header().Add("Cache-Control", cacheDuration)
	w.Header().Add("Content-Disposition", fmt.Sprintf("inline; filename=%s", baseName))

	http.ServeContent(w, r, baseName, item.Date, reader)
}

// List return all thumbnails in a base64 form
func (a App) List(w http.ResponseWriter, r *http.Request, item absto.Item, items []absto.Item) {
	if len(items) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	ctx, end := tracer.StartSpan(r.Context(), a.tracer, "thumbnail_list", trace.WithSpanKind(trace.SpanKindInternal))
	defer end()

	var hash string

	if query.GetBool(r, "search") {
		hash = a.thumbnailHash(ctx, items)
	} else if thumbnails, err := a.ListDir(ctx, item); err != nil {
		logger.WithField("item", item.Pathname).Error("list thumbnails: %s", err)
	} else {
		hash = sha.New(thumbnails)
	}

	etag, ok := provider.EtagMatch(w, r, hash)
	if ok {
		return
	}

	w.Header().Add("Content-Type", "text/plain; charset=utf-8")
	w.Header().Add("Cache-Control", "no-cache")
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
		a.encodeContent(ctx, w, isDone, item)
	}
}

func (a App) thumbnailHash(ctx context.Context, items []absto.Item) string {
	ctx, end := tracer.StartSpan(ctx, a.tracer, "thumbnail_hash", trace.WithSpanKind(trace.SpanKindInternal))
	defer end()

	ids := make([]string, len(items))
	for index, item := range items {
		ids[index] = a.PathForScale(item, SmallSize)
	}

	thumbnails, err := a.cacheApp.List(ctx, onCacheError, ids...)
	if err != nil {
		logger.Error("list thumbnails from cache: %s", err)
	}

	hasher := sha.Stream()

	for _, thumbnail := range thumbnails {
		hasher.Write(thumbnail)
	}

	return hasher.Sum()
}

func (a App) encodeContent(ctx context.Context, w io.Writer, isDone func() bool, item absto.Item) {
	if item.IsDir || isDone() {
		return
	}

	ctx, end := tracer.StartSpan(ctx, a.tracer, "thumbnail_encode", trace.WithSpanKind(trace.SpanKindInternal))
	defer end()

	reader, err := a.storageApp.ReadFrom(ctx, a.PathForScale(item, SmallSize))
	if err != nil {
		if !absto.IsNotExist(err) {
			logEncodeContentError(item).Error("open: %s", err)
		}

		return
	}
	defer provider.LogClose(reader, "thumbnail.encodeContent", item.Pathname)

	provider.DoneWriter(isDone, w, item.ID)
	provider.DoneWriter(isDone, w, `,`)

	buffer := provider.BufferPool.Get().(*bytes.Buffer)
	defer provider.BufferPool.Put(buffer)

	encoder := base64.NewEncoder(base64.StdEncoding, w)

	if _, err = io.CopyBuffer(encoder, reader, buffer.Bytes()); err != nil {
		if !absto.IsNotExist(a.storageApp.ConvertError(err)) {
			logEncodeContentError(item).Error("copy: %s", err)
		}
	}

	if err := encoder.Close(); err != nil {
		logger.Error("close thumbnail encoder: %s", err)
	}

	provider.DoneWriter(isDone, w, "\n")
}

func logEncodeContentError(item absto.Item) logger.Provider {
	return logger.WithField("fn", "thumbnail.encodeContent").WithField("item", item.Pathname)
}
