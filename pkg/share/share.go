package share

import (
	"context"
	"flag"
	"fmt"
	"strings"
	"sync"
	"syscall"
	"time"

	absto "github.com/ViBiOh/absto/pkg/model"
	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/flags"
	"github.com/ViBiOh/httputils/v4/pkg/amqp"
	"github.com/ViBiOh/httputils/v4/pkg/clock"
	"github.com/ViBiOh/httputils/v4/pkg/cron"
	"github.com/ViBiOh/httputils/v4/pkg/logger"
)

var (
	shareFilename     = provider.MetadataDirectoryName + "/shares.json"
	semaphoreDuration = time.Second * 10
)

// App of package
type App struct {
	storageApp absto.Storage
	shares     map[string]provider.Share
	clock      clock.Clock

	amqpClient              *amqp.Client
	amqpExchange            string
	amqpRoutingKey          string
	amqpExclusiveRoutingKey string

	sync.RWMutex
}

// Config of package
type Config struct {
	amqpExchange            *string
	amqpRoutingKey          *string
	amqpExclusiveRoutingKey *string
}

// Flags adds flags for configuring package
func Flags(fs *flag.FlagSet, prefix string) Config {
	return Config{
		amqpExchange:            flags.String(fs, prefix, "share", "AmqpExchange", "AMQP Exchange Name", "fibr.shares", nil),
		amqpRoutingKey:          flags.String(fs, prefix, "share", "AmqpRoutingKey", "AMQP Routing Key for share", "share", nil),
		amqpExclusiveRoutingKey: flags.String(fs, prefix, "share", "AmqpExclusiveRoutingKey", "AMQP Routing Key for exclusive lock on default exchange", "fibr.semaphore.shares", nil),
	}
}

// New creates new App from Config
func New(config Config, storageApp absto.Storage, amqpClient *amqp.Client) (*App, error) {
	var amqpExchange string
	var amqpExclusiveRoutingKey string

	if amqpClient != nil {
		amqpExchange = strings.TrimSpace(*config.amqpExchange)
		amqpExclusiveRoutingKey = strings.TrimSpace(*config.amqpExclusiveRoutingKey)

		if err := amqpClient.Publisher(amqpExchange, "fanout", nil); err != nil {
			return &App{}, fmt.Errorf("configure amqp: %s", err)
		}

		amqpExchange = strings.TrimSpace(*config.amqpExchange)
		if err := amqpClient.SetupExclusive(amqpExclusiveRoutingKey); err != nil {
			return &App{}, fmt.Errorf("setup amqp exclusive: %s", err)
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

// Exclusive does action on shares with exclusive lock
func (a *App) Exclusive(ctx context.Context, name string, duration time.Duration, action func(ctx context.Context) error) (bool, error) {
	fn := func() error {
		a.Lock()
		defer a.Unlock()

		if err := a.refresh(ctx); err != nil {
			return fmt.Errorf("refresh shares: %s", err)
		}

		return action(ctx)
	}

	if a.amqpClient == nil {
		return true, fn()
	}

exclusive:
	acquired, err := a.amqpClient.Exclusive(ctx, name, duration, func(ctx context.Context) error {
		return fn()
	})
	if err != nil {
		return true, err
	}
	if !acquired {
		time.Sleep(time.Second)
		goto exclusive
	}

	return true, nil
}

// Get returns a share based on path
func (a *App) Get(requestPath string) provider.Share {
	cleanPath := strings.TrimPrefix(requestPath, "/")

	a.RLock()
	defer a.RUnlock()

	for key, share := range a.shares {
		if strings.HasPrefix(cleanPath, key) {
			return share
		}
	}

	return provider.Share{}
}

// Start worker
func (a *App) Start(done <-chan struct{}) {
	if err := a.loadShares(context.Background()); err != nil {
		logger.Error("refresh shares: %s", err)
		return
	}

	cron := cron.New().Each(time.Hour).OnError(func(err error) {
		logger.Error("purge shares: %s", err)
	}).OnSignal(syscall.SIGUSR1)

	if a.amqpClient != nil {
		cron.Exclusive(a, a.amqpExclusiveRoutingKey, semaphoreDuration)
	}

	cron.Start(a.cleanShares, done)
}

func (a *App) loadShares(ctx context.Context) error {
	a.Lock()
	defer a.Unlock()

	return a.refresh(ctx)
}

func (a *App) refresh(ctx context.Context) error {
	if err := provider.LoadJSON(ctx, a.storageApp, shareFilename, &a.shares); err != nil {
		if !absto.IsNotExist(err) {
			return err
		}

		if err := a.storageApp.CreateDir(ctx, provider.MetadataDirectoryName); err != nil {
			return fmt.Errorf("create dir: %s", err)
		}

		return provider.SaveJSON(ctx, a.storageApp, shareFilename, &a.shares)
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
					logger.WithField("fn", "share.purgeExpiredShares").WithField("item", id).Error("publish share purge: %s", err)
				}
			}

			changed = true
		}
	}

	return changed
}

func (a *App) cleanShares(ctx context.Context) error {
	if !a.purgeExpiredShares() {
		return nil
	}

	return provider.SaveJSON(ctx, a.storageApp, shareFilename, &a.shares)
}
