package webhook

import (
	"context"
	"flag"
	"fmt"
	"strings"
	"sync"
	"time"

	absto "github.com/ViBiOh/absto/pkg/model"
	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/fibr/pkg/thumbnail"
	"github.com/ViBiOh/flags"
	"github.com/ViBiOh/httputils/v4/pkg/amqp"
	"github.com/ViBiOh/httputils/v4/pkg/logger"
	prom "github.com/ViBiOh/httputils/v4/pkg/prometheus"
	"github.com/ViBiOh/httputils/v4/pkg/renderer"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	webhookFilename   = provider.MetadataDirectoryName + "/webhooks.json"
	semaphoreDuration = time.Second * 10
)

// App of package
type App struct {
	storageApp absto.Storage
	webhooks   map[string]provider.Webhook
	counter    *prometheus.CounterVec

	amqpClient              *amqp.Client
	amqpExchange            string
	amqpExclusiveRoutingKey string
	amqpRoutingKey          string

	hmacSecret []byte

	rendererApp  renderer.App
	thumbnailApp thumbnail.App

	sync.RWMutex
}

// Config of package
type Config struct {
	hmacSecret *string

	amqpExchange            *string
	amqpRoutingKey          *string
	amqpExclusiveRoutingKey *string
}

// Flags adds flags for configuring package
func Flags(fs *flag.FlagSet, prefix string) Config {
	return Config{
		hmacSecret: flags.String(fs, prefix, "webhook", "Secret", "Secret for HMAC Signature", "", nil),

		amqpExchange:            flags.String(fs, prefix, "webhook", "AmqpExchange", "AMQP Exchange Name", "fibr.webhooks", nil),
		amqpRoutingKey:          flags.String(fs, prefix, "webhook", "AmqpRoutingKey", "AMQP Routing Key for webhook", "webhook", nil),
		amqpExclusiveRoutingKey: flags.String(fs, prefix, "webhook", "AmqpExclusiveRoutingKey", "AMQP Routing Key for exclusive lock on default exchange", "fibr.semaphore.webhooks", nil),
	}
}

// New creates new App from Config
func New(config Config, storageApp absto.Storage, prometheusRegisterer prometheus.Registerer, amqpClient *amqp.Client, rendererApp renderer.App, thumbnailApp thumbnail.App) (*App, error) {
	var amqpExchange string
	var amqpExclusiveRoutingKey string

	if amqpClient != nil {
		amqpExchange = strings.TrimSpace(*config.amqpExchange)
		amqpExclusiveRoutingKey = strings.TrimSpace(*config.amqpExclusiveRoutingKey)

		if err := amqpClient.Publisher(amqpExchange, "fanout", nil); err != nil {
			return &App{}, fmt.Errorf("configure amqp: %s", err)
		}

		amqpExchange = strings.TrimSpace(*config.amqpExchange)
		if err := amqpClient.SetupExclusive(amqpExclusiveRoutingKey); err != nil {
			return &App{}, fmt.Errorf("setup amqp exclusive: %s", err)
		}
	}

	return &App{
		storageApp:   storageApp,
		rendererApp:  rendererApp,
		thumbnailApp: thumbnailApp,
		webhooks:     make(map[string]provider.Webhook),
		counter:      prom.CounterVec(prometheusRegisterer, "fibr", "webhook", "item", "code"),
		hmacSecret:   []byte(*config.hmacSecret),

		amqpClient:              amqpClient,
		amqpExchange:            amqpExchange,
		amqpRoutingKey:          strings.TrimSpace(*config.amqpRoutingKey),
		amqpExclusiveRoutingKey: amqpExclusiveRoutingKey,
	}, nil
}

// Exclusive does action on webhook with exclusive lock
func (a *App) Exclusive(ctx context.Context, name string, duration time.Duration, action func(ctx context.Context) error) error {
	fn := func() error {
		a.Lock()
		defer a.Unlock()

		if err := a.loadWebhooks(ctx); err != nil {
			return fmt.Errorf("refresh webhooks: %s", err)
		}

		return action(ctx)
	}

	if a.amqpClient == nil {
		return fn()
	}

exclusive:
	acquired, err := a.amqpClient.Exclusive(ctx, name, duration, func(ctx context.Context) error {
		return fn()
	})
	if err != nil {
		return err
	}
	if !acquired {
		time.Sleep(time.Second)
		goto exclusive
	}

	return nil
}

// Start worker
func (a *App) Start(_ <-chan struct{}) {
	a.Lock()
	defer a.Unlock()

	if err := a.loadWebhooks(context.Background()); err != nil {
		logger.Error("refresh webhooks: %s", err)
		return
	}
}

func (a *App) loadWebhooks(ctx context.Context) error {
	if err := provider.LoadJSON(ctx, a.storageApp, webhookFilename, &a.webhooks); err != nil {
		if !absto.IsNotExist(err) {
			return err
		}

		if err := a.storageApp.CreateDir(ctx, provider.MetadataDirectoryName); err != nil {
			return fmt.Errorf("create dir: %s", err)
		}

		return provider.SaveJSON(ctx, a.storageApp, webhookFilename, &a.webhooks)
	}

	return nil
}
