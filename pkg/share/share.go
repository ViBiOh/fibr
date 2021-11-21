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
	"github.com/ViBiOh/httputils/v4/pkg/amqp"
	"github.com/ViBiOh/httputils/v4/pkg/clock"
	"github.com/ViBiOh/httputils/v4/pkg/cron"
	"github.com/ViBiOh/httputils/v4/pkg/flags"
	"github.com/ViBiOh/httputils/v4/pkg/logger"
)

var (
	shareFilename     = path.Join(provider.MetadataDirectoryName, "shares.json")
	semaphoreDuration = time.Second * 10
)

// App of package
type App struct {
	storageApp provider.Storage
	shares     map[string]provider.Share
	clock      *clock.Clock

	amqpClient              *amqp.Client
	amqpExchange            string
	amqpRoutingKey          string
	amqpExclusiveRoutingKey string

	mutex sync.RWMutex
}

// Config of package
type Config struct {
	amqpExchange            *string
	amqpRoutingKey          *string
	amqpExclusiveRoutingKey *string

	share *bool
}

// Flags adds flags for configuring package
func Flags(fs *flag.FlagSet, prefix string) Config {
	return Config{
		share: flags.New(prefix, "share", "Enabled").Default(true, nil).Label("Enable sharing feature").ToBool(fs),

		amqpExchange:            flags.New(prefix, "share", "AmqpExchange").Default("fibr-shares", nil).Label("AMQP Exchange Name").ToString(fs),
		amqpRoutingKey:          flags.New(prefix, "share", "AmqpRoutingKey").Default("share", nil).Label("AMQP Routing Key for share").ToString(fs),
		amqpExclusiveRoutingKey: flags.New(prefix, "share", "AmqpExclusiveRoutingKey").Default("fibr.semaphore.shares", nil).Label("AMQP Routing Key for exclusive lock on default exchange").ToString(fs),
	}
}

// New creates new App from Config
func New(config Config, storageApp provider.Storage, amqpClient *amqp.Client) (*App, error) {
	if !*config.share {
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
		shares:     make(map[string]provider.Share),
		storageApp: storageApp,

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

// Exclusive does action on shares with exclusive lock
func (a *App) Exclusive(ctx context.Context, name string, duration time.Duration, action func(ctx context.Context) error) error {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	fn := func() error {
		if err := a.refresh(); err != nil {
			return fmt.Errorf("unable to refresh shares: %s", err)
		}

		return action(ctx)
	}

	if a.amqpClient == nil {
		return fn()
	}

	return a.amqpClient.Exclusive(ctx, name, duration, func(ctx context.Context) error {
		return fn()
	})
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

	cron := cron.New().Each(time.Hour).OnError(func(err error) {
		logger.Error("unable to purge shares: %s", err)
	}).OnSignal(syscall.SIGUSR1)

	if a.amqpClient != nil {
		cron.Exclusive(a, a.amqpExclusiveRoutingKey, semaphoreDuration)
	}

	cron.Start(a.cleanShares, done)
}

func (a *App) refresh() error {
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

	for id, share := range a.shares {
		if share.IsExpired(now) {
			delete(a.shares, id)

			if a.amqpClient != nil {
				if err := a.amqpClient.PublishJSON(provider.Share{ID: id}, a.amqpExchange, a.amqpRoutingKey); err != nil {
					logger.WithField("fn", "share.purgeExpiredShares").WithField("item", id).Error("unable to publish share purge: %s", err)
				}
			}

			changed = true
		}
	}

	return changed
}

func (a *App) cleanShares(_ context.Context) error {
	if !a.purgeExpiredShares() {
		return nil
	}

	return provider.SaveJSON(a.storageApp, shareFilename, &a.shares)
}
