package share

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"syscall"
	"time"

	absto "github.com/ViBiOh/absto/pkg/model"
	"github.com/ViBiOh/fibr/pkg/exclusive"
	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/flags"
	"github.com/ViBiOh/httputils/v4/pkg/cron"
	"github.com/ViBiOh/httputils/v4/pkg/redis"
	"go.opentelemetry.io/otel/trace"
)

type GetNow func() time.Time

var shareFilename = provider.MetadataDirectoryName + "/shares.json"

type Service struct {
	exclusive     exclusive.Service
	storage       absto.Storage
	redisClient   redis.Client
	done          chan struct{}
	shares        map[string]provider.Share
	clock         GetNow
	cron          *cron.Cron
	pubsubChannel string
	mutex         sync.RWMutex
}

type Config struct {
	PubsubChannel string
}

func Flags(fs *flag.FlagSet, prefix string) *Config {
	var config Config

	flags.New("PubSubChannel", "Channel name").Prefix(prefix).DocPrefix("share").StringVar(fs, &config.PubsubChannel, "fibr:shares-channel", nil)

	return &config
}

func New(config *Config, tracerProvider trace.TracerProvider, storageService absto.Storage, redisClient redis.Client, exclusiveService exclusive.Service) (*Service, error) {
	return &Service{
		clock:         time.Now,
		cron:          cron.New().WithTracerProvider(tracerProvider),
		shares:        make(map[string]provider.Share),
		done:          make(chan struct{}),
		storage:       storageService,
		exclusive:     exclusiveService,
		redisClient:   redisClient,
		pubsubChannel: config.PubsubChannel,
	}, nil
}

func (s *Service) Done() <-chan struct{} {
	return s.done
}

func (s *Service) Exclusive(ctx context.Context, name string, duration time.Duration, action func(ctx context.Context) error) (bool, error) {
	return s.exclusive.Try(ctx, "fibr:mutex:"+name, duration, func(ctx context.Context) error {
		s.mutex.Lock()
		defer s.mutex.Unlock()

		if err := s.refresh(ctx); err != nil {
			return fmt.Errorf("refresh shares: %w", err)
		}

		return action(ctx)
	})
}

func (s *Service) Get(requestPath string) provider.Share {
	cleanPath := strings.TrimPrefix(requestPath, "/")

	s.mutex.RLock()
	defer s.mutex.RUnlock()

	for key, share := range s.shares {
		if strings.HasPrefix(cleanPath, key) {
			return share
		}
	}

	return provider.Share{}
}

func (s *Service) Start(ctx context.Context) {
	defer close(s.done)

	if err := s.loadShares(ctx); err != nil {
		slog.LogAttrs(ctx, slog.LevelError, "refresh shares", slog.Any("error", err))
		return
	}

	go redis.SubscribeFor(ctx, s.redisClient, s.pubsubChannel, s.PubSubHandle)

	purgeCron := s.cron.Each(time.Hour).OnError(func(ctx context.Context, err error) {
		slog.LogAttrs(ctx, slog.LevelError, "purge shares", slog.Any("error", err))
	}).OnSignal(syscall.SIGUSR1)

	if s.redisClient.Enabled() {
		purgeCron.Exclusive(s, "purge", exclusive.Duration)
	}

	purgeCron.Start(ctx, s.cleanShares)

	<-ctx.Done()
}

func (s *Service) loadShares(ctx context.Context) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	return s.refresh(ctx)
}

func (s *Service) refresh(ctx context.Context) error {
	if shares, err := provider.LoadJSON[map[string]provider.Share](ctx, s.storage, shareFilename); err != nil {
		if !absto.IsNotExist(err) {
			return err
		}

		if err := s.storage.Mkdir(ctx, provider.MetadataDirectoryName, absto.DirectoryPerm); err != nil {
			return fmt.Errorf("create dir: %w", err)
		}

		return provider.SaveJSON(ctx, s.storage, shareFilename, &s.shares)
	} else {
		s.shares = shares
	}

	return nil
}

func (s *Service) purgeExpiredShares(ctx context.Context) bool {
	now := s.clock()
	changed := false

	for id, share := range s.shares {
		if share.IsExpired(now) {
			delete(s.shares, id)

			if err := s.redisClient.PublishJSON(ctx, s.pubsubChannel, provider.Share{ID: id}); err != nil {
				slog.LogAttrs(ctx, slog.LevelError, "publish share purge", slog.String("fn", "share.purgeExpiredShares"), slog.String("item", id), slog.Any("error", err))
			}

			changed = true
		}
	}

	return changed
}

func (s *Service) cleanShares(ctx context.Context) error {
	if !s.purgeExpiredShares(ctx) {
		return nil
	}

	return provider.SaveJSON(ctx, s.storage, shareFilename, &s.shares)
}
