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

type App struct {
	done            chan struct{}
	storageApp      absto.Storage
	exclusiveApp    exclusive.App
	pushEvent       provider.EventProducer
	renamer         Renamer
	sanitizeOnStart bool
}

type Config struct {
	sanitizeOnStart *bool
}

func Flags(fs *flag.FlagSet, prefix string, overrides ...flags.Override) Config {
	return Config{
		sanitizeOnStart: flags.New("SanitizeOnStart", "Sanitize on start").Prefix(prefix).DocPrefix("crud").Bool(fs, false, nil),
	}
}

func New(config Config, storageApp absto.Storage, exclusiveApp exclusive.App, renamer Renamer, pushEvent provider.EventProducer) App {
	return App{
		done:            make(chan struct{}),
		storageApp:      storageApp,
		exclusiveApp:    exclusiveApp,
		renamer:         renamer,
		pushEvent:       pushEvent,
		sanitizeOnStart: *config.sanitizeOnStart,
	}
}

func (a App) Done() <-chan struct{} {
	return a.done
}

func (a App) Start(ctx context.Context) {
	defer close(a.done)

	if err := a.exclusiveApp.Execute(ctx, "fibr:mutex:start", time.Hour, func(ctx context.Context) error {
		return a.start(ctx)
	}); err != nil {
		slog.Error("start", "err", err)
	}
}

func (a App) start(ctx context.Context) error {
	slog.Info("Starting startup check...")
	defer slog.Info("Ending startup check.")

	done := ctx.Done()

	var directories []absto.Item

	err := a.storageApp.Walk(ctx, "", func(item absto.Item) error {
		select {
		case <-done:
			return errors.New("server is shutting down")
		default:
		}

		item = a.sanitizeName(ctx, item)

		if item.IsDir() {
			directories = append(directories, item)
		} else {
			a.pushEvent(ctx, provider.NewStartEvent(ctx, item))
		}

		return nil
	})
	if err != nil {
		return err
	}

	for _, directory := range directories {
		a.pushEvent(ctx, provider.NewStartEvent(ctx, directory))
	}

	return nil
}

func (a App) sanitizeName(ctx context.Context, item absto.Item) absto.Item {
	name, err := provider.SanitizeName(item.Pathname, false)
	if err != nil {
		slog.Error("sanitize name", "err", err, "item", item.Pathname)
		return item
	}

	if name == item.Pathname {
		return item
	}

	if !a.sanitizeOnStart {
		slog.Info("File should be renamed", "pathname", item.Pathname, "name", name)
		return item
	}

	slog.Info("Renaming...", "pathname", item.Pathname, "name", name)

	renamedItem, err := a.renamer.DoRename(ctx, item.Pathname, name, item)
	if err != nil {
		slog.Error("rename", "err", err)
		return item
	}

	return renamedItem
}
