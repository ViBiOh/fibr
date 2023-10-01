package thumbnail

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"path"
	"strings"
	"time"

	absto "github.com/ViBiOh/absto/pkg/model"
	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/flags"
	"github.com/ViBiOh/httputils/v4/pkg/amqp"
	"github.com/ViBiOh/httputils/v4/pkg/cache"
	"github.com/ViBiOh/httputils/v4/pkg/hash"
	"github.com/ViBiOh/httputils/v4/pkg/httperror"
	"github.com/ViBiOh/httputils/v4/pkg/query"
	"github.com/ViBiOh/httputils/v4/pkg/redis"
	"github.com/ViBiOh/httputils/v4/pkg/request"
	"github.com/ViBiOh/httputils/v4/pkg/telemetry"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

const SmallSize uint64 = 150

var cacheDuration = fmt.Sprintf("private, max-age=%.0f", (time.Minute * 5).Seconds())

type Service struct {
	redisClient   redis.Client
	tracer        trace.Tracer
	storage       absto.Storage
	smallStorage  absto.Storage
	largeStorage  absto.Storage
	pathnameInput chan absto.Item
	metric        metric.Int64Counter

	cache *cache.Cache[string, absto.Item]

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
	VithURL  string
	VithUser string
	VithPass string

	AmqpExchange            string
	AmqpStreamRoutingKey    string
	AmqpThumbnailRoutingKey string

	MaxSize      int64
	MinBitrate   uint64
	DirectAccess bool

	LargeSize uint64
}

func Flags(fs *flag.FlagSet, prefix string) *Config {
	var config Config

	flags.New("URL", "Vith Thumbnail URL").Prefix(prefix).DocPrefix("thumbnail").StringVar(fs, &config.VithURL, "http://vith:1080", nil)
	flags.New("User", "Vith Thumbnail Basic Auth User").Prefix(prefix).DocPrefix("thumbnail").StringVar(fs, &config.VithUser, "", nil)
	flags.New("Password", "Vith Thumbnail Basic Auth Password").Prefix(prefix).DocPrefix("thumbnail").StringVar(fs, &config.VithPass, "", nil)

	flags.New("DirectAccess", "Use Vith with direct access to filesystem (no large file upload, send a GET request, Basic Auth recommended)").Prefix(prefix).DocPrefix("thumbnail").BoolVar(fs, &config.DirectAccess, false, nil)
	flags.New("MaxSize", "Maximum file size (in bytes) for generating thumbnail (0 to no limit). Not used if DirectAccess enabled.").Prefix(prefix).DocPrefix("thumbnail").Int64Var(fs, &config.MaxSize, 1024*1024*200, nil)
	flags.New("MinBitrate", "Minimal video bitrate (in bits per second) to generate a streamable version (in HLS), if DirectAccess enabled").Prefix(prefix).DocPrefix("thumbnail").Uint64Var(fs, &config.MinBitrate, 80*1000*1000, nil)

	flags.New("AmqpExchange", "AMQP Exchange Name").Prefix(prefix).DocPrefix("thumbnail").StringVar(fs, &config.AmqpExchange, "fibr", nil)
	flags.New("AmqpStreamRoutingKey", "AMQP Routing Key for stream").Prefix(prefix).DocPrefix("thumbnail").StringVar(fs, &config.AmqpStreamRoutingKey, "stream", nil)
	flags.New("AmqpThumbnailRoutingKey", "AMQP Routing Key for thumbnail").Prefix(prefix).DocPrefix("thumbnail").StringVar(fs, &config.AmqpThumbnailRoutingKey, "thumbnail", nil)

	flags.New("LargeSize", "Size of large thumbnail for story display (thumbnail are always squared). 0 to disable").Prefix(prefix).DocPrefix("thumbnail").Uint64Var(fs, &config.LargeSize, 800, nil)

	return &config
}

func New(ctx context.Context, config *Config, storage absto.Storage, redisClient redis.Client, meterProvider metric.MeterProvider, traceProvider trace.TracerProvider, amqpClient *amqp.Client) (Service, error) {
	var amqpExchange string

	if amqpClient != nil {
		amqpExchange = config.AmqpExchange

		if err := amqpClient.Publisher(amqpExchange, "direct", nil); err != nil {
			return Service{}, fmt.Errorf("configure amqp: %w", err)
		}
	}

	var sizes []uint64
	if config.LargeSize > 0 {
		sizes = []uint64{SmallSize, config.LargeSize}
	} else {
		sizes = []uint64{SmallSize}
	}

	service := Service{
		vithRequest: request.New().URL(config.VithURL).BasicAuth(config.VithUser, config.VithPass).WithClient(provider.SlowClient),

		maxSize:      config.MaxSize,
		minBitrate:   config.MinBitrate,
		directAccess: config.DirectAccess,

		redisClient: redisClient,
		tracer:      traceProvider.Tracer("thumbnail"),

		amqpExchange:            amqpExchange,
		amqpStreamRoutingKey:    config.AmqpStreamRoutingKey,
		amqpThumbnailRoutingKey: config.AmqpThumbnailRoutingKey,

		storage: storage,
		smallStorage: storage.WithIgnoreFn(func(item absto.Item) bool {
			return !strings.HasSuffix(item.Name(), ".webp") || strings.HasSuffix(item.Name(), "_large.webp")
		}),
		largeStorage: storage.WithIgnoreFn(func(item absto.Item) bool {
			return !strings.HasSuffix(item.Name(), "_large.webp")
		}),
		amqpClient:    amqpClient,
		pathnameInput: make(chan absto.Item, provider.MaxConcurrency),

		largeSize: config.LargeSize,
		sizes:     sizes,
	}

	if meterProvider != nil {
		meter := meterProvider.Meter("github.com/ViBiOh/fibr/pkg/thumbnail")

		var err error

		service.metric, err = meter.Int64Counter("fibr_thumbnail")
		if err != nil {
			return service, fmt.Errorf("create thumbnail counter: %w", err)
		}
	}

	service.cache = cache.New(redisClient, redisKey, func(ctx context.Context, pathname string) (absto.Item, error) {
		return service.storage.Stat(ctx, pathname)
	}, traceProvider).
		WithMaxConcurrency(provider.MaxConcurrency).
		WithClientSideCaching(ctx, "fibr_thumbnail", provider.MaxClientSideCaching)

	return service, nil
}

func (s Service) LargeThumbnailSize() uint64 {
	return s.largeSize
}

func (s Service) Stream(w http.ResponseWriter, r *http.Request, item absto.Item) {
	reader, err := s.storage.ReadFrom(r.Context(), getStreamPath(item))
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

func (s Service) Serve(w http.ResponseWriter, r *http.Request, item absto.Item) {
	if !s.CanHaveThumbnail(item) {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	scale := SmallSize
	if rawScale := r.URL.Query().Get("scale"); len(rawScale) > 0 {
		if rawScale == "large" && s.largeSize > 0 {
			scale = s.largeSize
		}
	}

	ctx := r.Context()

	name := s.PathForScale(item, scale)

	reader, err := s.storage.ReadFrom(ctx, name)
	if err != nil {
		if absto.IsNotExist(err) {
			w.WriteHeader(http.StatusNoContent)
		}

		httperror.InternalServerError(w, err)

		return
	}

	defer provider.LogClose(reader, "thumbnail.Serve", item.Pathname)

	w.Header().Add("Cache-Control", cacheDuration)
	w.Header().Add("Content-Disposition", fmt.Sprintf("inline; filename=%s", path.Base(name)))

	http.ServeContent(w, r, name, item.Date, reader)
}

func (s Service) List(w http.ResponseWriter, r *http.Request, item absto.Item, items []absto.Item) {
	if len(items) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	ctx, end := telemetry.StartSpan(r.Context(), s.tracer, "list", trace.WithSpanKind(trace.SpanKindInternal))
	defer end(nil)

	var hash string

	if query.GetBool(r, "search") {
		hash = s.thumbnailHash(ctx, items)
	} else if thumbnails, err := s.ListDir(ctx, item); err != nil {
		slog.Error("list thumbnails", "err", err, "item", item.Pathname)
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
		s.encodeContent(ctx, w, isDone, item)

		if ok {
			flusher.Flush()
		}
	}
}

func (s Service) thumbnailHash(ctx context.Context, items []absto.Item) string {
	ctx, end := telemetry.StartSpan(ctx, s.tracer, "hash", trace.WithSpanKind(trace.SpanKindInternal))
	defer end(nil)

	ids := make([]string, len(items))
	for index, item := range items {
		ids[index] = s.PathForScale(item, SmallSize)
	}

	thumbnails, err := s.cache.List(ctx, ids...)
	if err != nil && !absto.IsNotExist(err) {
		slog.Error("list thumbnails from cache", "err", err)
	}

	hasher := hash.Stream()

	for _, thumbnail := range thumbnails {
		hasher.Write(thumbnail)
	}

	return hasher.Sum()
}

func (s Service) encodeContent(ctx context.Context, w io.Writer, isDone func() bool, item absto.Item) {
	if item.IsDir() || isDone() {
		return
	}

	ctx, end := telemetry.StartSpan(ctx, s.tracer, "encode", trace.WithSpanKind(trace.SpanKindInternal))
	defer end(nil)

	reader, err := s.storage.ReadFrom(ctx, s.PathForScale(item, SmallSize))
	if err != nil {
		if !absto.IsNotExist(err) {
			logEncodeContentError(item).Error("open", "err", err)
		}

		return
	}
	defer provider.LogClose(reader, "thumbnail.encodeContent", item.Pathname)

	provider.DoneWriter(isDone, w, item.ID)
	provider.DoneWriter(isDone, w, `,`)

	buffer := provider.BufferPool.Get().(*bytes.Buffer)
	defer provider.BufferPool.Put(buffer)

	if _, err = io.CopyBuffer(w, reader, buffer.Bytes()); err != nil {
		if !absto.IsNotExist(s.storage.ConvertError(err)) {
			logEncodeContentError(item).Error("copy", "err", err)
		}
	}

	provider.DoneWriter(isDone, w, "\x1c\x17\x04\x1c")
}

func logEncodeContentError(item absto.Item) *slog.Logger {
	return slog.With("fn", "thumbnail.encodeContent").With("item", item.Pathname)
}
