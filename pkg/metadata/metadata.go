package metadata

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
	metadataFilename = path.Join(provider.MetadataDirectoryName, ".json")
)

// App of package
type App interface {
	Enabled() bool
	GetShare(string) provider.Share
	CreateShare(string, bool, string, bool, time.Duration) (string, error)
	RenameSharePath(string, string) error
	DeleteShare(string) error
	DeleteSharePath(string) error
	Dump() map[string]provider.Share
	Start(<-chan struct{})
}

// Config of package
type Config struct {
	metadata *bool
}

type app struct {
	metadatas  map[string]provider.Share
	clock      *Clock
	storageApp provider.Storage
	mutex      sync.RWMutex
}

// Flags adds flags for configuring package
func Flags(fs *flag.FlagSet, prefix string, overrides ...flags.Override) Config {
	return Config{
		metadata: flags.New(prefix, "metadata").Name("Metadata").Default(flags.Default("Metadata", true, overrides)).Label("Enable metadata storage").ToBool(fs),
	}
}

// New creates new App from Config
func New(config Config, storageApp provider.Storage) App {
	if !*config.metadata {
		return &app{}
	}

	return &app{
		metadatas:  make(map[string]provider.Share),
		storageApp: storageApp,
	}
}

// GetShare returns share configuration if request path match
func (a *app) Enabled() bool {
	return a.metadatas != nil
}

// GetShare returns share configuration if request path match
func (a *app) GetShare(requestPath string) provider.Share {
	if !a.Enabled() {
		return provider.NoneShare
	}

	cleanPath := strings.TrimPrefix(requestPath, "/")

	a.mutex.RLock()
	defer a.mutex.RUnlock()

	for key, share := range a.metadatas {
		if strings.HasPrefix(cleanPath, key) {
			return share
		}
	}

	return provider.NoneShare
}

func (a *app) Dump() map[string]provider.Share {
	if !a.Enabled() {
		return nil
	}

	a.mutex.RLock()
	defer a.mutex.RUnlock()

	return a.metadatas
}

func (a *app) Start(done <-chan struct{}) {
	if !a.Enabled() {
		return
	}

	if err := a.refreshMetadatas(); err != nil {
		logger.Error("unable to refresh metadatas: %s", err)
		return
	}

	cron.New().Each(time.Hour).OnError(func(err error) {
		logger.Error("unable to purge metadatas: %s", err)
	}).OnSignal(syscall.SIGUSR1).Now().Start(a.cleanMetadatas, done)
}

func (a *app) refreshMetadatas() error {
	_, err := a.storageApp.Info(metadataFilename)
	if err != nil && !provider.IsNotExist(err) {
		return err
	}

	if provider.IsNotExist(err) {
		if err := a.storageApp.CreateDir(provider.MetadataDirectoryName); err != nil {
			return err
		}

		return nil
	}

	file, err := a.storageApp.ReaderFrom(metadataFilename)
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

	a.mutex.Lock()
	defer a.mutex.Unlock()

	decoder := json.NewDecoder(file)
	if err = decoder.Decode(&a.metadatas); err != nil {
		return fmt.Errorf("unable to decode metadatas: %s", err)
	}

	return nil
}

func (a *app) purgeExpiredMetadatas() bool {
	now := a.clock.Now()
	changed := false

	for key, share := range a.metadatas {
		if share.IsExpired(now) {
			delete(a.metadatas, key)
			changed = true
		}
	}

	return changed
}

func (a *app) cleanMetadatas(_ context.Context) error {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	if !a.purgeExpiredMetadatas() {
		return nil
	}

	if err := a.saveMetadatas(); err != nil {
		return fmt.Errorf("unable to save metadatas: %s", err)
	}

	return nil
}

func (a *app) saveMetadatas() (err error) {
	file, err := a.storageApp.WriterTo(metadataFilename)
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

	if err := encoder.Encode(a.metadatas); err != nil {
		return fmt.Errorf("unable to json encode metadatas: %s", err)
	}

	return nil
}
