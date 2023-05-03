package sanitizer

import (
	"context"
	"errors"
	"flag"
	"time"

	absto "github.com/ViBiOh/absto/pkg/model"
	"github.com/ViBiOh/fibr/pkg/exclusive"
	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/flags"
	"github.com/ViBiOh/httputils/v4/pkg/logger"
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
		logger.Error("start: %s", err)
	}
}

func (a App) start(ctx context.Context) error {
	logger.Info("Starting startup check...")
	defer logger.Info("Ending startup check.")

	done := ctx.Done()

	var directories []absto.Item

	err := a.storageApp.Walk(ctx, "", func(item absto.Item) error {
		select {
		case <-done:
			return errors.New("server is shutting down")
		default:
		}

		item = a.sanitizeName(ctx, item)

		if item.IsDir {
			directories = append(directories, item)
		} else {
			a.pushEvent(provider.NewStartEvent(ctx, item))
		}

		return nil
	})
	if err != nil {
		return err
	}

	for _, directory := range directories {
		a.pushEvent(provider.NewStartEvent(ctx, directory))
	}

	return nil
}

func (a App) sanitizeName(ctx context.Context, item absto.Item) absto.Item {
	name, err := provider.SanitizeName(item.Pathname, false)
	if err != nil {
		logger.WithField("item", item.Pathname).Error("sanitize name: %s", err)
		return item
	}

	if name == item.Pathname {
		return item
	}

	if !a.sanitizeOnStart {
		logger.Info("File with name `%s` should be renamed to `%s`", item.Pathname, name)
		return item
	}

	logger.Info("Renaming `%s` to `%s`", item.Pathname, name)

	renamedItem, err := a.renamer.DoRename(ctx, item.Pathname, name, item)
	if err != nil {
		logger.Error("%s", err)
		return item
	}

	return renamedItem
}
