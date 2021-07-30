package share

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"path"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/httputils/v4/pkg/cron"
	"github.com/ViBiOh/httputils/v4/pkg/flags"
	"github.com/ViBiOh/httputils/v4/pkg/logger"
)

var (
	shareFilename = path.Join(provider.MetadataDirectoryName, ".json")
)

// App of package
type App interface {
	Enabled() bool
	Get(string) provider.Share
	Create(string, bool, string, bool, time.Duration) (string, error)
	RenamePath(string, string) error
	Delete(string) error
	DeletePath(string) error
	List() map[string]provider.Share
	Start(<-chan struct{})
}

// Config of package
type Config struct {
	share *bool
}

type app struct {
	shares     map[string]provider.Share
	clock      *Clock
	storageApp provider.Storage
	mutex      sync.RWMutex
}

// Flags adds flags for configuring package
func Flags(fs *flag.FlagSet, prefix string, overrides ...flags.Override) Config {
	return Config{
		share: flags.New(prefix, "share").Name("Share").Default(flags.Default("Share", true, overrides)).Label("Enable sharing feature").ToBool(fs),
	}
}

// New creates new App from Config
func New(config Config, storageApp provider.Storage) App {
	if !*config.share {
		return &app{}
	}

	return &app{
		shares:     make(map[string]provider.Share),
		storageApp: storageApp,
	}
}

// GetShare returns share configuration if request path match
func (a *app) Enabled() bool {
	return a.shares != nil
}

// GetShare returns share configuration if request path match
func (a *app) Get(requestPath string) provider.Share {
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

func (a *app) List() map[string]provider.Share {
	if !a.Enabled() {
		return nil
	}

	a.mutex.RLock()
	defer a.mutex.RUnlock()

	return a.shares
}

func (a *app) Start(done <-chan struct{}) {
	if !a.Enabled() {
		return
	}

	if err := a.refreshMetadatas(); err != nil {
		logger.Error("unable to refresh shares: %s", err)
		return
	}

	cron.New().Each(time.Hour).OnError(func(err error) {
		logger.Error("unable to purge shares: %s", err)
	}).OnSignal(syscall.SIGUSR1).Now().Start(a.cleanShares, done)
}

func (a *app) refreshMetadatas() error {
	_, err := a.storageApp.Info(shareFilename)
	if err != nil && !provider.IsNotExist(err) {
		return err
	}

	if provider.IsNotExist(err) {
		if err := a.storageApp.CreateDir(provider.MetadataDirectoryName); err != nil {
			return err
		}

		return nil
	}

	file, err := a.storageApp.ReaderFrom(shareFilename)
	if file != nil {
		defer func() {
			if err := file.Close(); err != nil {
				logger.Error("unable to close share file: %s", err)
			}
		}()
	}
	if err != nil {
		return err
	}

	a.mutex.Lock()
	defer a.mutex.Unlock()

	if err = json.NewDecoder(file).Decode(&a.shares); err != nil {
		return fmt.Errorf("unable to decode: %s", err)
	}

	return nil
}

func (a *app) purgeExpiredShares() bool {
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

func (a *app) cleanShares(_ context.Context) error {
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

func (a *app) saveShares() (err error) {
	file, err := a.storageApp.WriterTo(shareFilename)
	if file != nil {
		defer func() {
			if closeErr := file.Close(); closeErr != nil {
				err = fmt.Errorf("%s: %w", err, closeErr)
			}
		}()
	}
	if err != nil {
		return err
	}

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")

	if err := encoder.Encode(a.shares); err != nil {
		return fmt.Errorf("unable to encode: %s", err)
	}

	return nil
}
