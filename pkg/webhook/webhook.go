package webhook

import (
	"encoding/json"
	"flag"
	"fmt"
	"path"
	"strings"
	"sync"

	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/httputils/v4/pkg/flags"
	"github.com/ViBiOh/httputils/v4/pkg/logger"
	"github.com/ViBiOh/httputils/v4/pkg/request"
	"golang.org/x/net/context"
)

var (
	webhookFilename = path.Join(provider.MetadataDirectoryName, "webhooks.json")
)

// App of package
type App struct {
	webhooks   []provider.Webhook
	storageApp provider.Storage
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
func New(config Config, storageApp provider.Storage) App {
	if !*config.enabled {
		return App{}
	}

	return App{
		storageApp: storageApp,
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

	if err := a.loadWebhooks(); err != nil {
		logger.Error("unable to refresh webhooks: %s", err)
		return
	}
}

// EventConsumer handle event pushed to the event bus
func (a *App) EventConsumer(e provider.Event) {
	if !a.Enabled() {
		return
	}

	a.mutex.RLock()
	defer a.mutex.RUnlock()

	pathname := e.Item.Pathname
	newPathname := e.New.Pathname
	if !e.Item.IsDir {
		pathname = path.Dir(pathname)
		newPathname = path.Dir(newPathname)
	}

	for _, webhook := range a.webhooks {
		if !strings.EqualFold(webhook.Pathname, pathname) && !strings.EqualFold(webhook.Pathname, newPathname) {
			continue
		}

		req := request.New().Post(webhook.URL).ContentJSON()
		for key, val := range webhook.Headers {
			req.Header(key, val)
		}

		resp, err := req.JSON(context.Background(), e)
		if err != nil {
			logger.Error("error while sending webhook: %s", err)
			continue
		}

		if err := resp.Body.Close(); err != nil {
			logger.Error("unable to close response body: %s", err)
		}
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

	a.mutex.Lock()
	defer a.mutex.Unlock()

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

	a.mutex.RLock()
	defer a.mutex.RUnlock()

	if err := encoder.Encode(a.webhooks); err != nil {
		return fmt.Errorf("unable to encode: %s", err)
	}

	return nil
}
