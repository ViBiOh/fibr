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
	"github.com/prometheus/client_golang/prometheus"
)

var (
	webhookFilename = path.Join(provider.MetadataDirectoryName, "webhooks.json")
)

// App of package
type App struct {
	storageApp provider.Storage
	webhooks   map[string]provider.Webhook
	counter    *prometheus.CounterVec
	mutex      sync.RWMutex
}

// Config of package
type Config struct {
	enabled *bool
}

// Flags adds flags for configuring package
func Flags(fs *flag.FlagSet, prefix string, overrides ...flags.Override) Config {
	return Config{
		enabled: flags.New(prefix, "webhook").Name("Webhook").Default(flags.Default("Webhook", true, overrides)).Label("Enable webhook feature").ToBool(fs),
	}
}

// New creates new App from Config
func New(config Config, storageApp provider.Storage) *App {
	if !*config.enabled {
		return &App{}
	}

	return &App{
		storageApp: storageApp,
		webhooks:   make(map[string]provider.Webhook),
	}
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

	a.mutex.Lock()
	defer a.mutex.Unlock()

	if err := a.loadWebhooks(); err != nil {
		logger.Error("unable to refresh webhooks: %s", err)
		return
	}
}

func (a *App) loadWebhooks() (err error) {
	file, err := a.storageApp.ReaderFrom(webhookFilename)
	if err != nil {
		if provider.IsNotExist(err) {
			return a.saveWebhooks()
		}
		return err
	}

	defer func() {
		if err := file.Close(); err != nil {
			logger.Error("unable to close webhook file: %s", err)
		}
	}()

	if err = json.NewDecoder(file).Decode(&a.webhooks); err != nil {
		return fmt.Errorf("unable to decode: %s", err)
	}

	return nil
}

func (a *App) saveWebhooks() (err error) {
	file, err := a.storageApp.WriterTo(webhookFilename)
	if err != nil {
		return err
	}

	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			err = fmt.Errorf("%s: %w", err, closeErr)
		}
	}()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")

	if err := encoder.Encode(a.webhooks); err != nil {
		return fmt.Errorf("unable to encode: %s", err)
	}

	return nil
}
