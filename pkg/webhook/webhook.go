package webhook

import (
	"context"
	"flag"
	"fmt"
	"strings"
	"sync"

	absto "github.com/ViBiOh/absto/pkg/model"
	"github.com/ViBiOh/fibr/pkg/exclusive"
	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/fibr/pkg/thumbnail"
	"github.com/ViBiOh/flags"
	"github.com/ViBiOh/httputils/v4/pkg/cntxt"
	"github.com/ViBiOh/httputils/v4/pkg/logger"
	prom "github.com/ViBiOh/httputils/v4/pkg/prometheus"
	"github.com/ViBiOh/httputils/v4/pkg/redis"
	"github.com/ViBiOh/httputils/v4/pkg/renderer"
	"github.com/prometheus/client_golang/prometheus"
)

var webhookFilename = provider.MetadataDirectoryName + "/webhooks.json"

type App struct {
	exclusiveApp  exclusive.App
	storageApp    absto.Storage
	done          chan struct{}
	webhooks      map[string]provider.Webhook
	counter       *prometheus.CounterVec
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

func New(config Config, storageApp absto.Storage, prometheusRegisterer prometheus.Registerer, redisClient redis.Client, rendererApp *renderer.App, thumbnailApp thumbnail.App, exclusiveApp exclusive.App) *App {
	return &App{
		done:          make(chan struct{}),
		storageApp:    storageApp,
		rendererApp:   rendererApp,
		thumbnailApp:  thumbnailApp,
		exclusiveApp:  exclusiveApp,
		webhooks:      make(map[string]provider.Webhook),
		counter:       prom.CounterVec(prometheusRegisterer, "fibr", "webhook", "item", "code"),
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
		logger.Error("refresh webhooks: %s", err)
		return
	}

	done, unsubscribe := redis.SubscribeFor(ctx, a.redisClient, a.pubsubChannel, a.PubSubHandle)
	defer func() { <-done }()
	defer func() {
		logger.Info("Unsubscribing Webhook's PubSub...")

		if unsubscribeErr := unsubscribe(cntxt.WithoutDeadline(ctx)); unsubscribeErr != nil {
			logger.Error("Webhook's unsubscribe: %s", unsubscribeErr)
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

		if err := a.storageApp.Mkdir(ctx, provider.MetadataDirectoryName, provider.DirectoryPerm); err != nil {
			return fmt.Errorf("create dir: %w", err)
		}

		return provider.SaveJSON(ctx, a.storageApp, webhookFilename, &a.webhooks)
	} else {
		a.webhooks = webhooks
	}

	return nil
}
