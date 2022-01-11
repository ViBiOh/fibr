package webhook

import (
	"context"
	"flag"
	"fmt"
	"path"
	"strings"
	"sync"
	"time"

	absto "github.com/ViBiOh/absto/pkg/model"
	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/fibr/pkg/thumbnail"
	"github.com/ViBiOh/httputils/v4/pkg/amqp"
	"github.com/ViBiOh/httputils/v4/pkg/flags"
	"github.com/ViBiOh/httputils/v4/pkg/logger"
	prom "github.com/ViBiOh/httputils/v4/pkg/prometheus"
	"github.com/ViBiOh/httputils/v4/pkg/renderer"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	webhookFilename   = path.Join(provider.MetadataDirectoryName, "webhooks.json")
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
		hmacSecret: flags.New(prefix, "webhook", "Secret").Default("", nil).Label("Secret for HMAC Signature").ToString(fs),

		amqpExchange:            flags.New(prefix, "webhook", "AmqpExchange").Default("fibr.webhooks", nil).Label("AMQP Exchange Name").ToString(fs),
		amqpRoutingKey:          flags.New(prefix, "webhook", "AmqpRoutingKey").Default("webhook", nil).Label("AMQP Routing Key for webhook").ToString(fs),
		amqpExclusiveRoutingKey: flags.New(prefix, "webhook", "AmqpExclusiveRoutingKey").Default("fibr.semaphore.webhooks", nil).Label("AMQP Routing Key for exclusive lock on default exchange").ToString(fs),
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
			return &App{}, fmt.Errorf("unable to configure amqp: %s", err)
		}

		amqpExchange = strings.TrimSpace(*config.amqpExchange)
		if err := amqpClient.SetupExclusive(amqpExclusiveRoutingKey); err != nil {
			return &App{}, fmt.Errorf("unable to setup amqp exclusive: %s", err)
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
	a.Lock()
	defer a.Unlock()

	fn := func() error {
		if err := a.loadWebhooks(); err != nil {
			return fmt.Errorf("unable to refresh webhooks: %s", err)
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
	if err := a.loadWebhooks(); err != nil {
		logger.Error("unable to refresh webhooks: %s", err)
		return
	}
}

func (a *App) loadWebhooks() error {
	if err := provider.LoadJSON(a.storageApp, webhookFilename, &a.webhooks); err != nil {
		if !absto.IsNotExist(err) {
			return err
		}

		if err := a.storageApp.CreateDir(provider.MetadataDirectoryName); err != nil {
			return fmt.Errorf("unable to create dir: %s", err)
		}

		return provider.SaveJSON(a.storageApp, webhookFilename, &a.webhooks)
	}

	return nil
}
