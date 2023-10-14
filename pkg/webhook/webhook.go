package webhook

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"sync"

	absto "github.com/ViBiOh/absto/pkg/model"
	"github.com/ViBiOh/fibr/pkg/exclusive"
	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/fibr/pkg/thumbnail"
	"github.com/ViBiOh/flags"
	"github.com/ViBiOh/httputils/v4/pkg/cntxt"
	"github.com/ViBiOh/httputils/v4/pkg/redis"
	"github.com/ViBiOh/httputils/v4/pkg/renderer"
	"go.opentelemetry.io/otel/metric"
)

var webhookFilename = provider.MetadataDirectoryName + "/webhooks.json"

type Service struct {
	exclusiveService exclusive.Service
	storage          absto.Storage
	done             chan struct{}
	webhooks         map[string]provider.Webhook
	counter          metric.Int64Counter
	redisClient      redis.Client
	pubsubChannel    string
	rendererService  *renderer.Service
	hmacSecret       []byte
	thumbnail        thumbnail.Service
	mutex            sync.RWMutex
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

func New(config *Config, storageService absto.Storage, meterProvider metric.MeterProvider, redisClient redis.Client, rendererService *renderer.Service, thumbnailService thumbnail.Service, exclusiveApp exclusive.Service) *Service {
	var counter metric.Int64Counter
	if meterProvider != nil {
		meter := meterProvider.Meter("github.com/ViBiOh/fibr/pkg/webhook")

		var err error

		counter, err = meter.Int64Counter("fibr.webhook")
		if err != nil {
			slog.Error("create webhook counter", "err", err)
		}
	}

	return &Service{
		done:             make(chan struct{}),
		storage:          storageService,
		rendererService:  rendererService,
		thumbnail:        thumbnailService,
		exclusiveService: exclusiveApp,
		webhooks:         make(map[string]provider.Webhook),
		counter:          counter,
		hmacSecret:       []byte(config.HmacSecret),
		redisClient:      redisClient,
		pubsubChannel:    config.PubsubChannel,
	}
}

func (s *Service) Done() <-chan struct{} {
	return s.done
}

func (s *Service) Exclusive(ctx context.Context, name string, action func(ctx context.Context) error) error {
	return s.exclusiveService.Execute(ctx, "fibr:mutex:"+name, exclusive.Duration, func(ctx context.Context) error {
		if err := s.loadWebhooks(ctx); err != nil {
			return fmt.Errorf("refresh webhooks: %w", err)
		}

		return action(ctx)
	})
}

func (s *Service) Start(ctx context.Context) {
	defer close(s.done)

	if err := s.loadWebhooks(ctx); err != nil {
		slog.Error("refresh webhooks", "err", err)
		return
	}

	unsubscribe := redis.SubscribeFor(ctx, s.redisClient, s.pubsubChannel, s.PubSubHandle)
	defer func() {
		slog.Info("Unsubscribing Webhook's PubSub...")

		if unsubscribeErr := unsubscribe(cntxt.WithoutDeadline(ctx)); unsubscribeErr != nil {
			slog.Error("Webhook's unsubscribe", "err", unsubscribeErr)
		}
	}()

	<-ctx.Done()
}

func (s *Service) loadWebhooks(ctx context.Context) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if webhooks, err := provider.LoadJSON[map[string]provider.Webhook](ctx, s.storage, webhookFilename); err != nil {
		if !absto.IsNotExist(err) {
			return err
		}

		if err := s.storage.Mkdir(ctx, provider.MetadataDirectoryName, absto.DirectoryPerm); err != nil {
			return fmt.Errorf("create dir: %w", err)
		}

		return provider.SaveJSON(ctx, s.storage, webhookFilename, &s.webhooks)
	} else {
		s.webhooks = webhooks
	}

	return nil
}
