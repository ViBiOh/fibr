package share

import (
	"context"
	"flag"
	"fmt"
	"path"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/httputils/v4/pkg/clock"
	"github.com/ViBiOh/httputils/v4/pkg/cron"
	"github.com/ViBiOh/httputils/v4/pkg/flags"
	"github.com/ViBiOh/httputils/v4/pkg/logger"
)

var shareFilename = path.Join(provider.MetadataDirectoryName, "shares.json")

// App of package
type App struct {
	shares     map[string]provider.Share
	clock      *clock.Clock
	storageApp provider.Storage
	mutex      sync.RWMutex
}

// Config of package
type Config struct {
	share *bool
}

// Flags adds flags for configuring package
func Flags(fs *flag.FlagSet, prefix string) Config {
	return Config{
		share: flags.New(prefix, "share", "Share").Default(true, nil).Label("Enable sharing feature").ToBool(fs),
	}
}

// New creates new App from Config
func New(config Config, storageApp provider.Storage) *App {
	if !*config.share {
		return &App{}
	}

	return &App{
		shares:     make(map[string]provider.Share),
		storageApp: storageApp,
	}
}

// Enabled checks if requirements are met
func (a *App) Enabled() bool {
	return a.storageApp != nil
}

// Get returns a share based on path
func (a *App) Get(requestPath string) provider.Share {
	if !a.Enabled() {
		return provider.NoneShare
	}

	cleanPath := strings.TrimPrefix(requestPath, "/")

	a.mutex.RLock()
	defer a.mutex.RUnlock()

	for key, share := range a.shares {
		if strings.HasPrefix(cleanPath, key) {
			return share
		}
	}

	return provider.NoneShare
}

// Start worker
func (a *App) Start(done <-chan struct{}) {
	if !a.Enabled() {
		return
	}

	if err := a.refresh(); err != nil {
		logger.Error("unable to refresh shares: %s", err)
		return
	}

	cron.New().Each(time.Hour).OnError(func(err error) {
		logger.Error("unable to purge shares: %s", err)
	}).OnSignal(syscall.SIGUSR1).Now().Start(a.cleanShares, done)
}

func (a *App) refresh() error {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	if err := provider.LoadJSON(a.storageApp, shareFilename, &a.shares); err != nil {
		if !provider.IsNotExist(err) {
			return err
		}

		if err := a.storageApp.CreateDir(provider.MetadataDirectoryName); err != nil {
			return fmt.Errorf("unable to create dir: %s", err)
		}

		return provider.SaveJSON(a.storageApp, shareFilename, &a.shares)
	}

	return nil
}

func (a *App) purgeExpiredShares() bool {
	now := a.clock.Now()
	changed := false

	for key, share := range a.shares {
		if share.IsExpired(now) {
			delete(a.shares, key)
			changed = true
		}
	}

	return changed
}

func (a *App) cleanShares(_ context.Context) error {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	if !a.purgeExpiredShares() {
		return nil
	}

	if err := provider.SaveJSON(a.storageApp, shareFilename, &a.shares); err != nil {
		return fmt.Errorf("unable to save: %s", err)
	}

	return nil
}
