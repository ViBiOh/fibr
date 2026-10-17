package share

import (
	"context"
	"flag"
	"log/slog"
	"strings"
	"syscall"
	"time"

	absto "github.com/ViBiOh/absto/pkg/model"
	"github.com/ViBiOh/fibr/pkg/exclusive"
	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/fibr/pkg/registry"
	"github.com/ViBiOh/flags"
	"github.com/ViBiOh/httputils/v4/pkg/cron"
	"github.com/ViBiOh/httputils/v4/pkg/redis"
	"go.opentelemetry.io/otel/trace"
)

type GetNow func() time.Time

var shareFilename = provider.MetadataDirectoryName + "/shares.json"

type Service struct {
	shares        *registry.Registry[provider.Share]
	redisClient   redis.Client
	done          chan struct{}
	clock         GetNow
	cron          *cron.Cron
	pubsubChannel string
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
		shares:        registry.New[provider.Share](storageService, exclusiveService, redisClient, shareFilename, config.PubsubChannel),
		done:          make(chan struct{}),
		redisClient:   redisClient,
		pubsubChannel: config.PubsubChannel,
	}, nil
}

func (s *Service) Done() <-chan struct{} {
	return s.done
}

func (s *Service) Get(requestPath string) provider.Share {
	id, _, _ := strings.Cut(strings.TrimPrefix(requestPath, "/"), "/")

	return s.shares.Items()[id]
}

func (s *Service) Start(ctx context.Context) {
	defer close(s.done)

	if err := s.shares.Load(ctx); err != nil {
		slog.LogAttrs(ctx, slog.LevelError, "refresh shares", slog.Any("error", err))
		return
	}

	go redis.SubscribeFor(ctx, s.redisClient, s.pubsubChannel, s.PubSubHandle)

	purgeCron := s.cron.Each(time.Hour).OnError(func(ctx context.Context, err error) {
		slog.LogAttrs(ctx, slog.LevelError, "purge shares", slog.Any("error", err))
	}).OnSignal(syscall.SIGUSR1)

	purgeCron.Start(ctx, s.cleanShares)

	<-ctx.Done()
}

func (s *Service) cleanShares(ctx context.Context) error {
	return s.shares.Update(ctx, func(shares map[string]provider.Share) ([]provider.Share, error) {
		return purgeExpired(shares, s.clock()), nil
	})
}

func purgeExpired(shares map[string]provider.Share, now time.Time) []provider.Share {
	var purged []provider.Share

	for id, share := range shares {
		if share.IsExpired(now) {
			delete(shares, id)

			purged = append(purged, provider.Share{ID: id})
		}
	}

	return purged
}
