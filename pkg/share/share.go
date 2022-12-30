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
	"github.com/ViBiOh/httputils/v4/pkg/cron"
	"github.com/ViBiOh/httputils/v4/pkg/logger"
)

type GetNow func() time.Time

var shareFilename = provider.MetadataDirectoryName + "/shares.json"

type App struct {
	storageApp absto.Storage
	shares     map[string]provider.Share
	clock      GetNow

	amqpClient              *amqp.Client
	amqpExchange            string
	amqpRoutingKey          string
	amqpExclusiveRoutingKey string

	mutex sync.RWMutex
}

type Config struct {
	amqpExchange            *string
	amqpRoutingKey          *string
	amqpExclusiveRoutingKey *string
}

func Flags(fs *flag.FlagSet, prefix string) Config {
	return Config{
		amqpExchange:            flags.String(fs, prefix, "share", "AmqpExchange", "AMQP Exchange Name", "fibr.shares", nil),
		amqpRoutingKey:          flags.String(fs, prefix, "share", "AmqpRoutingKey", "AMQP Routing Key for share", "share", nil),
		amqpExclusiveRoutingKey: flags.String(fs, prefix, "share", "AmqpExclusiveRoutingKey", "AMQP Routing Key for exclusive lock on default exchange", "fibr.semaphore.shares", nil),
	}
}

func New(config Config, storageApp absto.Storage, amqpClient *amqp.Client) (*App, error) {
	var amqpExchange string
	var amqpExclusiveRoutingKey string

	if amqpClient != nil {
		amqpExchange = strings.TrimSpace(*config.amqpExchange)
		amqpExclusiveRoutingKey = strings.TrimSpace(*config.amqpExclusiveRoutingKey)

		if err := amqpClient.Publisher(amqpExchange, "fanout", nil); err != nil {
			return &App{}, fmt.Errorf("configure amqp: %w", err)
		}

		amqpExchange = strings.TrimSpace(*config.amqpExchange)
		if err := amqpClient.SetupExclusive(amqpExclusiveRoutingKey); err != nil {
			return &App{}, fmt.Errorf("setup amqp exclusive: %w", err)
		}
	}

	return &App{
		clock: time.Now,

		shares:     make(map[string]provider.Share),
		storageApp: storageApp,

		amqpClient:              amqpClient,
		amqpExchange:            amqpExchange,
		amqpRoutingKey:          strings.TrimSpace(*config.amqpRoutingKey),
		amqpExclusiveRoutingKey: amqpExclusiveRoutingKey,
	}, nil
}

func (a *App) Exclusive(ctx context.Context, name string, duration time.Duration, action func(ctx context.Context) error) (bool, error) {
	fn := func() error {
		a.mutex.Lock()
		defer a.mutex.Unlock()

		if err := a.refresh(ctx); err != nil {
			return fmt.Errorf("refresh shares: %w", err)
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

func (a *App) Get(requestPath string) provider.Share {
	cleanPath := strings.TrimPrefix(requestPath, "/")

	a.mutex.RLock()
	defer a.mutex.RUnlock()

	for key, share := range a.shares {
		if strings.HasPrefix(cleanPath, key) {
			return share
		}
	}

	return provider.Share{}
}

func (a *App) Start(ctx context.Context) {
	if err := a.loadShares(ctx); err != nil {
		logger.Error("refresh shares: %s", err)
		return
	}

	purgeCron := cron.New().Each(time.Hour).OnError(func(err error) {
		logger.Error("purge shares: %s", err)
	}).OnSignal(syscall.SIGUSR1)

	if a.amqpClient != nil {
		purgeCron.Exclusive(a, a.amqpExclusiveRoutingKey, provider.SemaphoreDuration)
	}

	purgeCron.Start(ctx, a.cleanShares)
}

func (a *App) loadShares(ctx context.Context) error {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	return a.refresh(ctx)
}

func (a *App) refresh(ctx context.Context) error {
	if shares, err := provider.LoadJSON[map[string]provider.Share](ctx, a.storageApp, shareFilename); err != nil {
		if !absto.IsNotExist(err) {
			return err
		}

		if err := a.storageApp.CreateDir(ctx, provider.MetadataDirectoryName); err != nil {
			return fmt.Errorf("create dir: %w", err)
		}

		return provider.SaveJSON(ctx, a.storageApp, shareFilename, &a.shares)
	} else {
		a.shares = shares
	}

	return nil
}

func (a *App) purgeExpiredShares(ctx context.Context) bool {
	now := a.clock()
	changed := false

	for id, share := range a.shares {
		if share.IsExpired(now) {
			delete(a.shares, id)

			if a.amqpClient != nil {
				if err := a.amqpClient.PublishJSON(ctx, provider.Share{ID: id}, a.amqpExchange, a.amqpRoutingKey); err != nil {
					logger.WithField("fn", "share.purgeExpiredShares").WithField("item", id).Error("publish share purge: %s", err)
				}
			}

			changed = true
		}
	}

	return changed
}

func (a *App) cleanShares(ctx context.Context) error {
	if !a.purgeExpiredShares(ctx) {
		return nil
	}

	return provider.SaveJSON(ctx, a.storageApp, shareFilename, &a.shares)
}
