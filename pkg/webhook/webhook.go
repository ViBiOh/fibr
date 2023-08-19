package webhook

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"strings"
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

type App struct {
	exclusiveApp  exclusive.App
	storageApp    absto.Storage
	done          chan struct{}
	webhooks      map[string]provider.Webhook
	counter       metric.Int64Counter
	redisClient   redis.Client
	pubsubChannel string
	rendererApp   *renderer.App
	hmacSecret    []byte
	thumbnailApp  thumbnail.App
	mutex         sync.RWMutex
}

type Config struct {
	hmacSecret    *string
	pubsubChannel *string
}

func Flags(fs *flag.FlagSet, prefix string) Config {
	return Config{
		hmacSecret:    flags.New("Secret", "Secret for HMAC Signature").Prefix(prefix).DocPrefix("webhook").String(fs, "", nil),
		pubsubChannel: flags.New("PubSubChannel", "Channel name").Prefix(prefix).DocPrefix("share").String(fs, "fibr:webhooks-channel", nil),
	}
}

func New(config Config, storageApp absto.Storage, meterProvider metric.MeterProvider, redisClient redis.Client, rendererApp *renderer.App, thumbnailApp thumbnail.App, exclusiveApp exclusive.App) *App {
	var counter metric.Int64Counter
	if meterProvider != nil {
		meter := meterProvider.Meter("github.com/ViBiOh/fibr/pkg/webhook")

		var err error

		counter, err = meter.Int64Counter("fibr.webhook")
		if err != nil {
			slog.Error("create webhook counter", "err", err)
		}
	}

	return &App{
		done:          make(chan struct{}),
		storageApp:    storageApp,
		rendererApp:   rendererApp,
		thumbnailApp:  thumbnailApp,
		exclusiveApp:  exclusiveApp,
		webhooks:      make(map[string]provider.Webhook),
		counter:       counter,
		hmacSecret:    []byte(*config.hmacSecret),
		redisClient:   redisClient,
		pubsubChannel: strings.TrimSpace(*config.pubsubChannel),
	}
}

func (a *App) Done() <-chan struct{} {
	return a.done
}

func (a *App) Exclusive(ctx context.Context, name string, action func(ctx context.Context) error) error {
	return a.exclusiveApp.Execute(ctx, "fibr:mutex:"+name, exclusive.Duration, func(ctx context.Context) error {
		if err := a.loadWebhooks(ctx); err != nil {
			return fmt.Errorf("refresh webhooks: %w", err)
		}

		return action(ctx)
	})
}

func (a *App) Start(ctx context.Context) {
	defer close(a.done)

	if err := a.loadWebhooks(ctx); err != nil {
		slog.Error("refresh webhooks", "err", err)
		return
	}

	done, unsubscribe := redis.SubscribeFor(ctx, a.redisClient, a.pubsubChannel, a.PubSubHandle)
	defer func() { <-done }()
	defer func() {
		slog.Info("Unsubscribing Webhook's PubSub...")

		if unsubscribeErr := unsubscribe(cntxt.WithoutDeadline(ctx)); unsubscribeErr != nil {
			slog.Error("Webhook's unsubscribe", "err", unsubscribeErr)
		}
	}()

	<-ctx.Done()
}

func (a *App) loadWebhooks(ctx context.Context) error {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	if webhooks, err := provider.LoadJSON[map[string]provider.Webhook](ctx, a.storageApp, webhookFilename); err != nil {
		if !absto.IsNotExist(err) {
			return err
		}

		if err := a.storageApp.Mkdir(ctx, provider.MetadataDirectoryName, absto.DirectoryPerm); err != nil {
			return fmt.Errorf("create dir: %w", err)
		}

		return provider.SaveJSON(ctx, a.storageApp, webhookFilename, &a.webhooks)
	} else {
		a.webhooks = webhooks
	}

	return nil
}
