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
	"github.com/ViBiOh/httputils/v4/pkg/amqp"
	"github.com/ViBiOh/httputils/v4/pkg/logger"
	prom "github.com/ViBiOh/httputils/v4/pkg/prometheus"
	"github.com/ViBiOh/httputils/v4/pkg/renderer"
	"github.com/prometheus/client_golang/prometheus"
)

var webhookFilename = provider.MetadataDirectoryName + "/webhooks.json"

type App struct {
	exclusiveApp   exclusive.App
	storageApp     absto.Storage
	webhooks       map[string]provider.Webhook
	counter        *prometheus.CounterVec
	amqpClient     *amqp.Client
	amqpExchange   string
	amqpRoutingKey string
	rendererApp    renderer.App
	hmacSecret     []byte
	thumbnailApp   thumbnail.App
	mutex          sync.RWMutex
}

type Config struct {
	hmacSecret *string

	amqpExchange   *string
	amqpRoutingKey *string
}

func Flags(fs *flag.FlagSet, prefix string) Config {
	return Config{
		hmacSecret:     flags.String(fs, prefix, "webhook", "Secret", "Secret for HMAC Signature", "", nil),
		amqpExchange:   flags.String(fs, prefix, "webhook", "AmqpExchange", "AMQP Exchange Name", "fibr.webhooks", nil),
		amqpRoutingKey: flags.String(fs, prefix, "webhook", "AmqpRoutingKey", "AMQP Routing Key for webhook", "webhook", nil),
	}
}

func New(config Config, storageApp absto.Storage, prometheusRegisterer prometheus.Registerer, amqpClient *amqp.Client, rendererApp renderer.App, thumbnailApp thumbnail.App, exclusiveApp exclusive.App) (*App, error) {
	var amqpExchange string

	if amqpClient != nil {
		amqpExchange = strings.TrimSpace(*config.amqpExchange)

		if err := amqpClient.Publisher(amqpExchange, "fanout", nil); err != nil {
			return &App{}, fmt.Errorf("configure amqp: %w", err)
		}
	}

	return &App{
		storageApp:     storageApp,
		rendererApp:    rendererApp,
		thumbnailApp:   thumbnailApp,
		exclusiveApp:   exclusiveApp,
		webhooks:       make(map[string]provider.Webhook),
		counter:        prom.CounterVec(prometheusRegisterer, "fibr", "webhook", "item", "code"),
		hmacSecret:     []byte(*config.hmacSecret),
		amqpClient:     amqpClient,
		amqpExchange:   amqpExchange,
		amqpRoutingKey: strings.TrimSpace(*config.amqpRoutingKey),
	}, nil
}

func (a *App) Exclusive(ctx context.Context, name string, action func(ctx context.Context) error) error {
	return a.exclusiveApp.Execute(ctx, "fibr:mutex:"+name, exclusive.Duration, func(ctx context.Context) error {
		a.mutex.Lock()
		defer a.mutex.Unlock()

		if err := a.loadWebhooks(ctx); err != nil {
			return fmt.Errorf("refresh webhooks: %w", err)
		}

		return action(ctx)
	})
}

func (a *App) Start(ctx context.Context) {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	if err := a.loadWebhooks(ctx); err != nil {
		logger.Error("refresh webhooks: %s", err)
		return
	}
}

func (a *App) loadWebhooks(ctx context.Context) error {
	if webhooks, err := provider.LoadJSON[map[string]provider.Webhook](ctx, a.storageApp, webhookFilename); err != nil {
		if !absto.IsNotExist(err) {
			return err
		}

		if err := a.storageApp.CreateDir(ctx, provider.MetadataDirectoryName); err != nil {
			return fmt.Errorf("create dir: %w", err)
		}

		return provider.SaveJSON(ctx, a.storageApp, webhookFilename, &a.webhooks)
	} else {
		a.webhooks = webhooks
	}

	return nil
}
