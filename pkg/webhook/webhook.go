package webhook

import (
	"context"
	"flag"
	"fmt"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/httputils/v4/pkg/amqp"
	"github.com/ViBiOh/httputils/v4/pkg/flags"
	"github.com/ViBiOh/httputils/v4/pkg/logger"
	prom "github.com/ViBiOh/httputils/v4/pkg/prometheus"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	webhookFilename   = path.Join(provider.MetadataDirectoryName, "webhooks.json")
	semaphoreDuration = time.Second * 10
)

// App of package
type App struct {
	storageApp provider.Storage
	webhooks   map[string]provider.Webhook
	counter    *prometheus.CounterVec
	hmacSecret []byte
	mutex      sync.RWMutex

	amqpClient              *amqp.Client
	amqpExchange            string
	amqpRoutingKey          string
	amqpExclusiveRoutingKey string
}

// Config of package
type Config struct {
	enabled    *bool
	hmacSecret *string

	amqpExchange            *string
	amqpRoutingKey          *string
	amqpExclusiveRoutingKey *string
}

// Flags adds flags for configuring package
func Flags(fs *flag.FlagSet, prefix string) Config {
	return Config{
		enabled:    flags.New(prefix, "webhook", "Enabled").Default(true, nil).Label("Enable webhook feature").ToBool(fs),
		hmacSecret: flags.New(prefix, "webhook", "Secret").Default("", nil).Label("Secret for HMAC Signature").ToString(fs),

		amqpExchange:            flags.New(prefix, "webhook", "AmqpExchange").Default("fibr-webhooks", nil).Label("AMQP Exchange Name").ToString(fs),
		amqpRoutingKey:          flags.New(prefix, "webhook", "AmqpRoutingKey").Default("webhook", nil).Label("AMQP Routing Key for webhook").ToString(fs),
		amqpExclusiveRoutingKey: flags.New(prefix, "webhook", "AmqpExclusiveRoutingKey").Default("fibr.semaphore.webhooks", nil).Label("AMQP Routing Key for exclusive lock on default exchange").ToString(fs),
	}
}

// New creates new App from Config
func New(config Config, storageApp provider.Storage, prometheusRegisterer prometheus.Registerer, amqpClient *amqp.Client) (*App, error) {
	if !*config.enabled {
		return &App{}, nil
	}

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
		storageApp: storageApp,
		webhooks:   make(map[string]provider.Webhook),
		counter:    prom.CounterVec(prometheusRegisterer, "fibr", "webhook", "item", "code"),
		hmacSecret: []byte(*config.hmacSecret),

		amqpClient:              amqpClient,
		amqpExchange:            amqpExchange,
		amqpRoutingKey:          strings.TrimSpace(*config.amqpRoutingKey),
		amqpExclusiveRoutingKey: amqpExclusiveRoutingKey,
	}, nil
}

// Enabled checks if requirements are met
func (a *App) Enabled() bool {
	return a.storageApp != nil
}

// Exclusive does action on webhook with exclusive lock
func (a *App) Exclusive(ctx context.Context, name string, duration time.Duration, action func(ctx context.Context) error) error {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	if a.amqpClient == nil {
		if err := a.loadWebhooks(); err != nil {
			return fmt.Errorf("unable to load webhooks: %s", err)
		}

		return action(ctx)
	}

	return a.amqpClient.Exclusive(ctx, name, duration, func(ctx context.Context) error {
		if err := a.loadWebhooks(); err != nil {
			return fmt.Errorf("unable to load webhooks: %s", err)
		}

		return action(ctx)
	})
}

// Start worker
func (a *App) Start(_ <-chan struct{}) {
	if !a.Enabled() {
		return
	}

	if err := a.loadWebhooks(); err != nil {
		logger.Error("unable to refresh webhooks: %s", err)
		return
	}
}

func (a *App) loadWebhooks() error {
	if err := provider.LoadJSON(a.storageApp, webhookFilename, &a.webhooks); err != nil {
		if !provider.IsNotExist(err) {
			return err
		}

		if err := a.storageApp.CreateDir(provider.MetadataDirectoryName); err != nil {
			return fmt.Errorf("unable to create dir: %s", err)
		}

		return provider.SaveJSON(a.storageApp, webhookFilename, &a.webhooks)
	}

	return nil
}
