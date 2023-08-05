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
	"github.com/ViBiOh/fibr/pkg/exclusive"
	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/flags"
	"github.com/ViBiOh/httputils/v4/pkg/cntxt"
	"github.com/ViBiOh/httputils/v4/pkg/cron"
	"github.com/ViBiOh/httputils/v4/pkg/logger"
	"github.com/ViBiOh/httputils/v4/pkg/redis"
)

type GetNow func() time.Time

var shareFilename = provider.MetadataDirectoryName + "/shares.json"

type App struct {
	exclusiveApp  exclusive.App
	storageApp    absto.Storage
	redisClient   redis.Client
	done          chan struct{}
	shares        map[string]provider.Share
	clock         GetNow
	pubsubChannel string
	mutex         sync.RWMutex
}

type Config struct {
	pubsubChannel *string
}

func Flags(fs *flag.FlagSet, prefix string) Config {
	return Config{
		pubsubChannel: flags.New("PubSubChannel", "Channel name").Prefix(prefix).DocPrefix("share").String(fs, "fibr:shares-channel", nil),
	}
}

func New(config Config, storageApp absto.Storage, redisClient redis.Client, exclusiveApp exclusive.App) (*App, error) {
	return &App{
		clock:         time.Now,
		shares:        make(map[string]provider.Share),
		done:          make(chan struct{}),
		storageApp:    storageApp,
		exclusiveApp:  exclusiveApp,
		redisClient:   redisClient,
		pubsubChannel: strings.TrimSpace(*config.pubsubChannel),
	}, nil
}

func (a *App) Done() <-chan struct{} {
	return a.done
}

func (a *App) Exclusive(ctx context.Context, name string, duration time.Duration, action func(ctx context.Context) error) (bool, error) {
	return a.exclusiveApp.Try(ctx, "fibr:mutex:"+name, duration, func(ctx context.Context) error {
		a.mutex.Lock()
		defer a.mutex.Unlock()

		if err := a.refresh(ctx); err != nil {
			return fmt.Errorf("refresh shares: %w", err)
		}

		return action(ctx)
	})
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
	defer close(a.done)

	if err := a.loadShares(ctx); err != nil {
		logger.Error("refresh shares: %s", err)
		return
	}

	done, unsubscribe := redis.SubscribeFor(ctx, a.redisClient, a.pubsubChannel, a.PubSubHandle)
	defer func() { <-done }()
	defer func() {
		logger.Info("Unsubscribing Share's PubSub...")

		if unsubscribeErr := unsubscribe(cntxt.WithoutDeadline(ctx)); unsubscribeErr != nil {
			logger.Error("Share's unsubscribe: %s", unsubscribeErr)
		}
	}()

	purgeCron := cron.New().Each(time.Hour).OnError(func(err error) {
		logger.Error("purge shares: %s", err)
	}).OnSignal(syscall.SIGUSR1)

	if a.redisClient.Enabled() {
		purgeCron.Exclusive(a, "purge", exclusive.Duration)
	}

	purgeCron.Start(ctx, a.cleanShares)

	<-ctx.Done()
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

		if err := a.storageApp.Mkdir(ctx, provider.MetadataDirectoryName, absto.DirectoryPerm); err != nil {
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

			if err := a.redisClient.PublishJSON(ctx, a.pubsubChannel, provider.Share{ID: id}); err != nil {
				logger.WithField("fn", "share.purgeExpiredShares").WithField("item", id).Error("publish share purge: %s", err)
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
