package sanitizer

import (
	"context"
	"errors"
	"flag"
	"log/slog"
	"time"

	absto "github.com/ViBiOh/absto/pkg/model"
	"github.com/ViBiOh/fibr/pkg/exclusive"
	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/flags"
)

type Renamer interface {
	DoRename(ctx context.Context, oldPath, newPath string, oldItem absto.Item) (absto.Item, error)
}

type Service struct {
	done            chan struct{}
	storage         absto.Storage
	exclusive       exclusive.Service
	pushEvent       provider.EventProducer
	renamer         Renamer
	sanitizeOnStart bool
}

type Config struct {
	SanitizeOnStart bool
}

func Flags(fs *flag.FlagSet, prefix string, overrides ...flags.Override) *Config {
	var config Config

	flags.New("SanitizeOnStart", "Sanitize on start").Prefix(prefix).DocPrefix("crud").BoolVar(fs, &config.SanitizeOnStart, false, nil)

	return &config
}

func New(config *Config, storageService absto.Storage, exclusiveService exclusive.Service, renamer Renamer, pushEvent provider.EventProducer) Service {
	return Service{
		done:            make(chan struct{}),
		storage:         storageService,
		exclusive:       exclusiveService,
		renamer:         renamer,
		pushEvent:       pushEvent,
		sanitizeOnStart: config.SanitizeOnStart,
	}
}

func (s Service) Done() <-chan struct{} {
	return s.done
}

func (s Service) Start(ctx context.Context) {
	defer close(s.done)

	if err := s.exclusive.Execute(ctx, "fibr:mutex:start", time.Hour, func(ctx context.Context) error {
		return s.start(ctx)
	}); err != nil {
		slog.Error("start", "err", err)
	}
}

func (s Service) start(ctx context.Context) error {
	slog.Info("Starting startup check...")
	defer slog.Info("Ending startup check.")

	done := ctx.Done()

	var directories []absto.Item

	err := s.storage.Walk(ctx, "", func(item absto.Item) error {
		select {
		case <-done:
			return errors.New("server is shutting down")
		default:
		}

		item = s.sanitizeName(ctx, item)

		if item.IsDir() {
			directories = append(directories, item)
		} else {
			s.pushEvent(ctx, provider.NewStartEvent(ctx, item))
		}

		return nil
	})
	if err != nil {
		return err
	}

	for _, directory := range directories {
		s.pushEvent(ctx, provider.NewStartEvent(ctx, directory))
	}

	return nil
}

func (s Service) sanitizeName(ctx context.Context, item absto.Item) absto.Item {
	name, err := provider.SanitizeName(item.Pathname, false)
	if err != nil {
		slog.Error("sanitize name", "err", err, "item", item.Pathname)
		return item
	}

	if name == item.Pathname {
		return item
	}

	if !s.sanitizeOnStart {
		slog.Info("File should be renamed", "pathname", item.Pathname, "name", name)
		return item
	}

	slog.Info("Renaming...", "pathname", item.Pathname, "name", name)

	renamedItem, err := s.renamer.DoRename(ctx, item.Pathname, name, item)
	if err != nil {
		slog.Error("rename", "err", err)
		return item
	}

	return renamedItem
}
