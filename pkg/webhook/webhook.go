package webhook

import (
	"encoding/json"
	"flag"
	"fmt"
	"path"
	"sync"

	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/httputils/v4/pkg/flags"
	"github.com/ViBiOh/httputils/v4/pkg/logger"
	prom "github.com/ViBiOh/httputils/v4/pkg/prometheus"
	"github.com/prometheus/client_golang/prometheus"
)

var webhookFilename = path.Join(provider.MetadataDirectoryName, "webhooks.json")

// App of package
type App struct {
	storageApp provider.Storage
	webhooks   map[string]provider.Webhook
	counter    *prometheus.CounterVec
	hmacSecret []byte
	mutex      sync.RWMutex
}

// Config of package
type Config struct {
	enabled    *bool
	hmacSecret *string
}

// Flags adds flags for configuring package
func Flags(fs *flag.FlagSet, prefix string, overrides ...flags.Override) Config {
	return Config{
		enabled:    flags.New(prefix, "webhook", "Enabled").Default(true, overrides).Label("Enable webhook feature").ToBool(fs),
		hmacSecret: flags.New(prefix, "webhook", "Secret").Default("", overrides).Label("Secret for HMAC Signature").ToString(fs),
	}
}

// New creates new App from Config
func New(config Config, storageApp provider.Storage, prometheusRegisterer prometheus.Registerer) (*App, error) {
	if !*config.enabled {
		return &App{}, nil
	}

	return &App{
		storageApp: storageApp,
		webhooks:   make(map[string]provider.Webhook),
		counter:    prom.CounterVec(prometheusRegisterer, "fibr", "webhook", "item", "code"),
		hmacSecret: []byte(*config.hmacSecret),
	}, nil
}

// Enabled checks if requirements are met
func (a *App) Enabled() bool {
	return a.storageApp != nil
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
	file, err := a.storageApp.ReaderFrom(webhookFilename)
	if err != nil {
		if !provider.IsNotExist(err) {
			return fmt.Errorf("unable to read file: %s", err)
		}

		if err := a.storageApp.CreateDir(provider.MetadataDirectoryName); err != nil {
			return fmt.Errorf("unable to create directory: %s", err)
		}

		if err := provider.SaveJSON(a.storageApp, webhookFilename, a.webhooks); err != nil {
			return fmt.Errorf("unable to save file: %s", err)
		}
		return nil
	}

	a.mutex.Lock()
	defer a.mutex.Unlock()

	if err = json.NewDecoder(file).Decode(&a.webhooks); err != nil {
		err = fmt.Errorf("unable to decode: %s", err)
	}

	if closeErr := file.Close(); closeErr != nil {
		if err != nil {
			return fmt.Errorf("%s: %w", err, closeErr)
		}
		return fmt.Errorf("unable to close reader: %s", closeErr)
	}

	return err
}
