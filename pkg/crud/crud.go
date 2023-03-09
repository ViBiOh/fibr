package crud

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"strings"
	"time"

	absto "github.com/ViBiOh/absto/pkg/model"
	"github.com/ViBiOh/fibr/pkg/exclusive"
	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/fibr/pkg/search"
	"github.com/ViBiOh/fibr/pkg/thumbnail"
	"github.com/ViBiOh/flags"
	"github.com/ViBiOh/httputils/v4/pkg/logger"
	"github.com/ViBiOh/httputils/v4/pkg/renderer"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/crypto/bcrypt"
)

type Items []absto.Item

func (i Items) FindDirectory(name string) (absto.Item, bool) {
	for _, item := range i {
		if item.IsDir && absto.Dirname(item.Pathname) == name {
			return item, true
		}
	}

	return absto.Item{}, false
}

var (
	ErrNotAuthorized  = errors.New("you're not authorized to do this ⛔")
	ErrEmptyName      = errors.New("name is empty")
	ErrEmptyFolder    = errors.New("folder is empty")
	ErrAbsoluteFolder = errors.New("folder has to be absolute")
)

type App struct {
	tracer          trace.Tracer
	rawStorageApp   absto.Storage
	storageApp      absto.Storage
	shareApp        provider.ShareManager
	webhookApp      provider.WebhookManager
	metadataApp     provider.MetadataManager
	searchApp       search.App
	pushEvent       provider.EventProducer
	exclusiveApp    exclusive.App
	temporaryFolder string
	rendererApp     renderer.App
	thumbnailApp    thumbnail.App
	bcryptCost      int
	sanitizeOnStart bool
	chunkUpload     bool
}

type Config struct {
	bcryptDuration  *string
	temporaryFolder *string
	sanitizeOnStart *bool
	chunkUpload     *bool
}

func Flags(fs *flag.FlagSet, prefix string) Config {
	return Config{
		sanitizeOnStart: flags.Bool(fs, prefix, "crud", "SanitizeOnStart", "Sanitize name on start", false, nil),
		bcryptDuration:  flags.String(fs, prefix, "crud", "BcryptDuration", "Wanted bcrypt duration for calculating effective cost", "0.25s", nil),

		chunkUpload:     flags.Bool(fs, prefix, "crud", "ChunkUpload", "Use chunk upload in browser", false, nil),
		temporaryFolder: flags.String(fs, prefix, "crud", "TemporaryFolder", "Temporary folder for chunk upload", "/tmp", nil),
	}
}

func New(config Config, storageApp absto.Storage, filteredStorage absto.Storage, rendererApp renderer.App, shareApp provider.ShareManager, webhookApp provider.WebhookManager, thumbnailApp thumbnail.App, exifApp provider.MetadataManager, searchApp search.App, eventProducer provider.EventProducer, exclusiveApp exclusive.App, tracer trace.Tracer) (App, error) {
	app := App{
		sanitizeOnStart: *config.sanitizeOnStart,

		chunkUpload:     *config.chunkUpload,
		temporaryFolder: strings.TrimSpace(*config.temporaryFolder),

		tracer:    tracer,
		pushEvent: eventProducer,

		rawStorageApp: storageApp,
		storageApp:    filteredStorage,
		rendererApp:   rendererApp,
		thumbnailApp:  thumbnailApp,
		metadataApp:   exifApp,
		shareApp:      shareApp,
		webhookApp:    webhookApp,
		searchApp:     searchApp,

		exclusiveApp: exclusiveApp,
	}

	bcryptDuration, err := time.ParseDuration(strings.TrimSpace(*config.bcryptDuration))
	if err != nil {
		return app, fmt.Errorf("parse bcrypt duration: %w", err)
	}

	bcryptCost, err := findBcryptBestCost(bcryptDuration)
	if err != nil {
		logger.Error("find best bcrypt cost: %s", err)
		bcryptCost = bcrypt.DefaultCost
	}
	logger.Info("Best bcrypt cost is %d", bcryptCost)

	app.bcryptCost = bcryptCost

	return app, nil
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

	renamedItem, err := a.doRename(ctx, item.Pathname, name, item)
	if err != nil {
		logger.Error("%s", err)
		return item
	}

	return renamedItem
}

func (a App) error(w http.ResponseWriter, r *http.Request, request provider.Request, err error) {
	a.rendererApp.Error(w, r, map[string]any{"Request": request}, err)
}
