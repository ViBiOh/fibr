package thumbnail

import (
	"bytes"
	"context"
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
	"github.com/ViBiOh/httputils/v4/pkg/hash"
	"github.com/ViBiOh/httputils/v4/pkg/httperror"
	"github.com/ViBiOh/httputils/v4/pkg/logger"
	prom "github.com/ViBiOh/httputils/v4/pkg/prometheus"
	"github.com/ViBiOh/httputils/v4/pkg/query"
	"github.com/ViBiOh/httputils/v4/pkg/redis"
	"github.com/ViBiOh/httputils/v4/pkg/request"
	"github.com/ViBiOh/httputils/v4/pkg/tracer"
	"github.com/prometheus/client_golang/prometheus"
	"go.opentelemetry.io/otel/trace"
)

const SmallSize uint64 = 150

var cacheDuration = fmt.Sprintf("private, max-age=%.0f", (time.Minute * 5).Seconds())

type App struct {
	redisClient     redis.Client
	tracer          trace.Tracer
	storageApp      absto.Storage
	smallStorageApp absto.Storage
	largeStorageApp absto.Storage
	pathnameInput   chan absto.Item
	metric          *prometheus.CounterVec

	cacheApp *cache.App[string, absto.Item]

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
		vithURL:  flags.New("URL", "Vith Thumbnail URL").Prefix(prefix).DocPrefix("thumbnail").String(fs, "http://vith:1080", nil),
		vithUser: flags.New("User", "Vith Thumbnail Basic Auth User").Prefix(prefix).DocPrefix("thumbnail").String(fs, "", nil),
		vithPass: flags.New("Password", "Vith Thumbnail Basic Auth Password").Prefix(prefix).DocPrefix("thumbnail").String(fs, "", nil),

		directAccess: flags.New("DirectAccess", "Use Vith with direct access to filesystem (no large file upload, send a GET request, Basic Auth recommended)").Prefix(prefix).DocPrefix("thumbnail").Bool(fs, false, nil),
		maxSize:      flags.New("MaxSize", "Maximum file size (in bytes) for generating thumbnail (0 to no limit). Not used if DirectAccess enabled.").Prefix(prefix).DocPrefix("thumbnail").Int64(fs, 1024*1024*200, nil),
		minBitrate:   flags.New("MinBitrate", "Minimal video bitrate (in bits per second) to generate a streamable version (in HLS), if DirectAccess enabled").Prefix(prefix).DocPrefix("thumbnail").Uint64(fs, 80*1000*1000, nil),

		amqpExchange:            flags.New("AmqpExchange", "AMQP Exchange Name").Prefix(prefix).DocPrefix("thumbnail").String(fs, "fibr", nil),
		amqpStreamRoutingKey:    flags.New("AmqpStreamRoutingKey", "AMQP Routing Key for stream").Prefix(prefix).DocPrefix("thumbnail").String(fs, "stream", nil),
		amqpThumbnailRoutingKey: flags.New("AmqpThumbnailRoutingKey", "AMQP Routing Key for thumbnail").Prefix(prefix).DocPrefix("thumbnail").String(fs, "thumbnail", nil),

		largeSize: flags.New("LargeSize", "Size of large thumbnail for story display (thumbnail are always squared). 0 to disable").Prefix(prefix).DocPrefix("thumbnail").Uint64(fs, 800, nil),
	}
}

func New(config Config, storage absto.Storage, redisClient redis.Client, prometheusRegisterer prometheus.Registerer, tracerApp tracer.App, amqpClient *amqp.Client) (App, error) {
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
			return !strings.HasSuffix(item.Name(), ".webp") || strings.HasSuffix(item.Name(), "_large.webp")
		}),
		largeStorageApp: storage.WithIgnoreFn(func(item absto.Item) bool {
			return !strings.HasSuffix(item.Name(), "_large.webp")
		}),
		amqpClient:    amqpClient,
		metric:        prom.CounterVec(prometheusRegisterer, "fibr", "thumbnail", "item", "type", "state"),
		pathnameInput: make(chan absto.Item, provider.MaxConcurrency),

		largeSize: largeSize,
		sizes:     sizes,
	}

	app.cacheApp = cache.New(redisClient, redisKey, func(ctx context.Context, pathname string) (absto.Item, error) {
		return app.storageApp.Stat(ctx, pathname)
	}, redisCacheDuration, provider.MaxConcurrency, tracerApp.GetTracer("thumbnail_cache"))

	return app, nil
}

func (a App) LargeThumbnailSize() uint64 {
	return a.largeSize
}

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
	http.ServeContent(w, r, item.Name(), item.Date, reader)
}

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

func (a App) List(w http.ResponseWriter, r *http.Request, item absto.Item, items []absto.Item) {
	if len(items) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	ctx, end := tracer.StartSpan(r.Context(), a.tracer, "list", trace.WithSpanKind(trace.SpanKindInternal))
	defer end(nil)

	var hash string

	if query.GetBool(r, "search") {
		hash = a.thumbnailHash(ctx, items)
	} else if thumbnails, err := a.ListDir(ctx, item); err != nil {
		logger.WithField("item", item.Pathname).Error("list thumbnails: %s", err)
	} else {
		hash = provider.RawHash(thumbnails)
	}

	etag, ok := provider.EtagMatch(w, r, hash)
	if ok {
		return
	}

	w.Header().Add("Content-Type", "application/octet-stream")
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

	flusher, ok := w.(http.Flusher)

	for _, item := range items {
		a.encodeContent(ctx, w, isDone, item)

		if ok {
			flusher.Flush()
		}
	}
}

func (a App) thumbnailHash(ctx context.Context, items []absto.Item) string {
	ctx, end := tracer.StartSpan(ctx, a.tracer, "hash", trace.WithSpanKind(trace.SpanKindInternal))
	defer end(nil)

	ids := make([]string, len(items))
	for index, item := range items {
		ids[index] = a.PathForScale(item, SmallSize)
	}

	thumbnails, err := a.cacheApp.List(ctx, onCacheError, ids...)
	if err != nil && !absto.IsNotExist(err) {
		logger.Error("list thumbnails from cache: %s", err)
	}

	hasher := hash.Stream()

	for _, thumbnail := range thumbnails {
		hasher.Write(thumbnail)
	}

	return hasher.Sum()
}

func (a App) encodeContent(ctx context.Context, w io.Writer, isDone func() bool, item absto.Item) {
	if item.IsDir() || isDone() {
		return
	}

	ctx, end := tracer.StartSpan(ctx, a.tracer, "encode", trace.WithSpanKind(trace.SpanKindInternal))
	defer end(nil)

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

	if _, err = io.CopyBuffer(w, reader, buffer.Bytes()); err != nil {
		if !absto.IsNotExist(a.storageApp.ConvertError(err)) {
			logEncodeContentError(item).Error("copy: %s", err)
		}
	}

	provider.DoneWriter(isDone, w, "\x1c\x17\x04\x1c")
}

func logEncodeContentError(item absto.Item) logger.Provider {
	return logger.WithField("fn", "thumbnail.encodeContent").WithField("item", item.Pathname)
}
