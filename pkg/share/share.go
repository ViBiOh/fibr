package share

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
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

var (
	oldShareFilename = path.Join(provider.MetadataDirectoryName, ".json")
	shareFilename    = path.Join(provider.MetadataDirectoryName, "shares.json")
)

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
func Flags(fs *flag.FlagSet, prefix string, overrides ...flags.Override) Config {
	return Config{
		share: flags.New(prefix, "share", "Share").Default(true, overrides).Label("Enable sharing feature").ToBool(fs),
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

	if err := a.migrate(); err != nil {
		logger.Error("unable to migrate shares file: %s", err)
	}

	if err := a.refresh(); err != nil {
		logger.Error("unable to refresh shares: %s", err)
		return
	}

	cron.New().Each(time.Hour).OnError(func(err error) {
		logger.Error("unable to purge shares: %s", err)
	}).OnSignal(syscall.SIGUSR1).Now().Start(a.cleanShares, done)
}

func (a *App) migrate() error {
	_, err := a.storageApp.Info(oldShareFilename)
	if err != nil {
		if provider.IsNotExist(err) {
			return nil
		}

		return err
	}

	logger.Info("Migrating old share file to new one")

	oldFile, err := a.storageApp.ReaderFrom(oldShareFilename)
	if err != nil {
		return fmt.Errorf("unable to read from old file: %s", err)
	}

	defer func() {
		if err := oldFile.Close(); err != nil {
			logger.Error("unable to close old share file: %s", err)
		}
	}()

	newFile, err := a.storageApp.WriterTo(shareFilename)
	if err != nil {
		return fmt.Errorf("unable to write to new file: %s", err)
	}

	defer func() {
		if err := newFile.Close(); err != nil {
			logger.Error("unable to close new share file: %s", err)
		}
	}()

	buffer := provider.BufferPool.Get().(*bytes.Buffer)
	defer provider.BufferPool.Put(buffer)

	if _, err := io.CopyBuffer(newFile, oldFile, buffer.Bytes()); err != nil {
		return fmt.Errorf("unable to copy files: %s", err)
	}

	if err := a.storageApp.Remove(oldShareFilename); err != nil {
		return fmt.Errorf("unable to remove old file: %s", err)
	}

	return nil
}

func (a *App) refresh() error {
	file, err := a.storageApp.ReaderFrom(shareFilename)
	if err != nil {
		if !provider.IsNotExist(err) {
			return err
		}

		if err := a.storageApp.CreateDir(provider.MetadataDirectoryName); err != nil {
			return err
		}

		return a.saveShares()
	}

	defer func() {
		if err := file.Close(); err != nil {
			logger.Error("unable to close share file: %s", err)
		}
	}()

	a.mutex.Lock()
	defer a.mutex.Unlock()

	if err = json.NewDecoder(file).Decode(&a.shares); err != nil {
		return fmt.Errorf("unable to decode: %s", err)
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

	if err := a.saveShares(); err != nil {
		return fmt.Errorf("unable to save: %s", err)
	}

	return nil
}

func (a *App) saveShares() (err error) {
	file, err := a.storageApp.WriterTo(shareFilename)
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

	if err := encoder.Encode(a.shares); err != nil {
		return fmt.Errorf("unable to encode: %s", err)
	}

	return nil
}
