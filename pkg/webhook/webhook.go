package webhook

import (
	"context"
	"flag"
	"log/slog"
	"time"

	absto "github.com/ViBiOh/absto/pkg/model"
	"github.com/ViBiOh/fibr/pkg/exclusive"
	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/fibr/pkg/push"
	"github.com/ViBiOh/fibr/pkg/registry"
	"github.com/ViBiOh/fibr/pkg/thumbnail"
	"github.com/ViBiOh/flags"
	"github.com/ViBiOh/httputils/v4/pkg/redis"
	"github.com/ViBiOh/httputils/v4/pkg/renderer"
	"go.opentelemetry.io/otel/metric"
)

var webhookFilename = provider.MetadataDirectoryName + "/webhooks.json"

type Service struct {
	counter         metric.Int64Counter
	redisClient     redis.Client
	webhooks        *registry.Registry[provider.Webhook]
	debouncer       *GroupDebouncer[provider.Event]
	rendererService *renderer.Service
	push            *push.Service
	done            chan struct{}
	pubsubChannel   string
	hmacSecret      []byte
	thumbnail       thumbnail.Service
}

type Config struct {
	HmacSecret    string
	PubsubChannel string
}

func Flags(fs *flag.FlagSet, prefix string) *Config {
	var config Config

	flags.New("Secret", "Secret for HMAC Signature").Prefix(prefix).DocPrefix("webhook").StringVar(fs, &config.HmacSecret, "", nil)
	flags.New("PubSubChannel", "Channel name").Prefix(prefix).DocPrefix("share").StringVar(fs, &config.PubsubChannel, "fibr:webhooks-channel", nil)

	return &config
}

func New(config *Config, storageService absto.Storage, meterProvider metric.MeterProvider, redisClient redis.Client, rendererService *renderer.Service, pushService *push.Service, thumbnailService thumbnail.Service, exclusiveApp exclusive.Service) *Service {
	var counter metric.Int64Counter
	if meterProvider != nil {
		meter := meterProvider.Meter("github.com/ViBiOh/fibr/pkg/webhook")

		var err error

		counter, err = meter.Int64Counter("fibr.webhook")
		if err != nil {
			slog.LogAttrs(context.Background(), slog.LevelError, "create webhook counter", slog.Any("error", err))
		}
	}

	service := &Service{
		done:            make(chan struct{}),
		rendererService: rendererService,
		thumbnail:       thumbnailService,
		push:            pushService,
		webhooks:        registry.New[provider.Webhook](storageService, exclusiveApp, redisClient, webhookFilename, config.PubsubChannel),
		counter:         counter,
		hmacSecret:      []byte(config.HmacSecret),
		redisClient:     redisClient,
		pubsubChannel:   config.PubsubChannel,
	}

	service.debouncer = NewDebouncer(time.Minute*30, service.asyncPushNotification)

	return service
}

func (s *Service) Done() <-chan struct{} {
	done := make(chan struct{})

	go func() {
		defer close(done)

		<-s.done
		<-s.debouncer.Done()
	}()

	return done
}

func (s *Service) Start(ctx context.Context) {
	defer close(s.done)

	if err := s.webhooks.Load(ctx); err != nil {
		slog.LogAttrs(ctx, slog.LevelError, "refresh webhooks", slog.Any("error", err))
		return
	}

	go s.debouncer.Start(ctx)

	redis.SubscribeFor(ctx, s.redisClient, s.pubsubChannel, s.PubSubHandle)
}
