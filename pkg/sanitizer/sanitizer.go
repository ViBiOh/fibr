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
	storageApp      absto.Storage
	exclusiveApp    exclusive.App
	pushEvent       provider.EventProducer
	renamer         Renamer
	sanitizeOnStart bool
}

type Config struct {
	sanitizeOnStart *bool
}

type Items []absto.Item

func (i Items) FindDirectory(name string) (absto.Item, bool) {
	for _, item := range i {
		if item.IsDir && absto.Dirname(item.Pathname) == name {
			return item, true
		}
	}

	return absto.Item{}, false
}

func Flags(fs *flag.FlagSet, prefix string, overrides ...flags.Override) Config {
	return Config{
		sanitizeOnStart: flags.Bool(fs, prefix, "crud", "SanitizeOnStart", "Sanitize on start", false, nil),
	}
}

func New(config Config, storageApp absto.Storage, exclusiveApp exclusive.App, renamer Renamer, pushEvent provider.EventProducer) App {
	return App{
		storageApp:      storageApp,
		exclusiveApp:    exclusiveApp,
		renamer:         renamer,
		pushEvent:       pushEvent,
		sanitizeOnStart: *config.sanitizeOnStart,
	}
}

func (a App) Start(ctx context.Context) {
	if err := a.exclusiveApp.Execute(ctx, "fibr:mutex:start", time.Hour, func(ctx context.Context) error {
		a.start(ctx)
		return nil
	}); err != nil {
		logger.Error("start: %s", err)
	}
}

func (a App) start(ctx context.Context) {
	logger.Info("Starting startup check...")
	defer logger.Info("Ending startup check.")

	done := ctx.Done()

	var directories Items

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
			a.notify(ctx, provider.NewStartEvent(item))

			if created := a.sanitizeOrphan(ctx, directories, item); !created.IsZero() {
				directories = append(directories, created)
			}
		}

		return nil
	})
	if err != nil {
		logger.Error("start: %s", err)
	}

	for _, directory := range directories {
		a.notify(ctx, provider.NewStartEvent(directory))
	}
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

	return a.rename(ctx, item, name)
}

func (a App) sanitizeOrphan(ctx context.Context, directories Items, item absto.Item) absto.Item {
	dirname := item.Dir()

	_, ok := directories.FindDirectory(dirname)
	if ok {
		return absto.Item{}
	}

	if !a.sanitizeOnStart {
		logger.Warn("File with name `%s` doesn't have a parent directory", item.Pathname)
		return absto.Item{}
	}

	sanitizedName, err := provider.SanitizeName(dirname, false)
	if err != nil {
		logger.Error("sanitize name for directory `%s`: %s", dirname, err)
		return absto.Item{}
	}

	logger.Info("Creating folder `%s`", sanitizedName)

	if err := a.storageApp.CreateDir(ctx, sanitizedName); err != nil {
		logger.Error("create a parent directory for `%s`: %s", item.Pathname, err)
		return absto.Item{}
	}

	directoryItem, err := a.storageApp.Info(ctx, sanitizedName)
	if err != nil {
		logger.Error("getting the parent directory infos `%s`: %s", item.Pathname, err)
		return absto.Item{}
	}

	return directoryItem
}

func (a App) rename(ctx context.Context, item absto.Item, name string) absto.Item {
	logger.Info("Renaming `%s` to `%s`", item.Pathname, name)

	renamedItem, err := a.renamer.DoRename(ctx, item.Pathname, name, item)
	if err != nil {
		logger.Error("%s", err)
		return item
	}

	return renamedItem
}
