package crud

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	absto "github.com/ViBiOh/absto/pkg/model"
	"github.com/ViBiOh/fibr/pkg/exif"
	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/fibr/pkg/thumbnail"
	"github.com/ViBiOh/flags"
	"github.com/ViBiOh/httputils/v4/pkg/amqp"
	"github.com/ViBiOh/httputils/v4/pkg/logger"
	"github.com/ViBiOh/httputils/v4/pkg/renderer"
	"github.com/ViBiOh/httputils/v4/pkg/tracer"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrNotAuthorized  = errors.New("you're not authorized to do this ⛔")
	ErrEmptyName      = errors.New("name is empty")
	ErrEmptyFolder    = errors.New("folder is empty")
	ErrAbsoluteFolder = errors.New("folder has to be absolute")
)

type App struct {
	tracer        trace.Tracer
	rawStorageApp absto.Storage
	storageApp    absto.Storage
	shareApp      provider.ShareManager
	webhookApp    provider.WebhookManager
	exifApp       provider.ExifManager
	pushEvent     provider.EventProducer

	amqpClient              *amqp.Client
	amqpExclusiveRoutingKey string

	temporaryFolder string
	rendererApp     renderer.App
	thumbnailApp    thumbnail.App
	bcryptCost      int
	sanitizeOnStart bool
	chunkUpload     bool
}

type Config struct {
	ignore                  *string
	amqpExclusiveRoutingKey *string
	bcryptDuration          *string
	temporaryFolder         *string
	sanitizeOnStart         *bool
	chunkUpload             *bool
}

func Flags(fs *flag.FlagSet, prefix string) Config {
	return Config{
		ignore:          flags.String(fs, prefix, "crud", "IgnorePattern", "Ignore pattern when listing files or directory", "", nil),
		sanitizeOnStart: flags.Bool(fs, prefix, "crud", "SanitizeOnStart", "Sanitize name on start", false, nil),
		bcryptDuration:  flags.String(fs, prefix, "crud", "BcryptDuration", "Wanted bcrypt duration for calculating effective cost", "0.25s", nil),

		chunkUpload:     flags.Bool(fs, prefix, "crud", "ChunkUpload", "Use chunk upload in browser", false, nil),
		temporaryFolder: flags.String(fs, prefix, "crud", "TemporaryFolder", "Temporary folder for chunk upload", "/tmp", nil),

		amqpExclusiveRoutingKey: flags.String(fs, prefix, "crud", "AmqpExclusiveRoutingKey", "AMQP Routing Key for exclusive lock on default exchange", "fibr.semaphore.start", nil),
	}
}

func New(config Config, storage absto.Storage, rendererApp renderer.App, shareApp provider.ShareManager, webhookApp provider.WebhookManager, thumbnailApp thumbnail.App, exifApp exif.App, eventProducer provider.EventProducer, amqpClient *amqp.Client, tracerApp tracer.App) (App, error) {
	app := App{
		sanitizeOnStart: *config.sanitizeOnStart,

		chunkUpload:     *config.chunkUpload,
		temporaryFolder: strings.TrimSpace(*config.temporaryFolder),

		tracer:    tracerApp.GetTracer("crud"),
		pushEvent: eventProducer,

		rawStorageApp: storage,
		rendererApp:   rendererApp,
		thumbnailApp:  thumbnailApp,
		exifApp:       exifApp,
		shareApp:      shareApp,
		webhookApp:    webhookApp,

		amqpClient:              amqpClient,
		amqpExclusiveRoutingKey: strings.TrimSpace(*config.amqpExclusiveRoutingKey),
	}

	if amqpClient != nil {
		if err := amqpClient.SetupExclusive(app.amqpExclusiveRoutingKey); err != nil {
			return app, fmt.Errorf("setup amqp exclusive: %s", err)
		}
	}

	var ignorePattern *regexp.Regexp
	ignore := *config.ignore
	if len(ignore) != 0 {
		pattern, err := regexp.Compile(ignore)
		if err != nil {
			return App{}, err
		}

		ignorePattern = pattern
		logger.Info("Ignoring files with pattern `%s`", ignore)
	}

	app.storageApp = storage.WithIgnoreFn(func(item absto.Item) bool {
		if strings.HasPrefix(item.Pathname, provider.MetadataDirectoryName) {
			return true
		}

		if ignorePattern != nil && ignorePattern.MatchString(item.Name) {
			return true
		}

		return false
	})

	bcryptDuration, err := time.ParseDuration(strings.TrimSpace(*config.bcryptDuration))
	if err != nil {
		return app, fmt.Errorf("parse bcrypt duration: %s", err)
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

func (a App) Start(done <-chan struct{}) {
	if a.amqpClient == nil {
		a.start(context.Background(), done)
		return
	}

	if _, err := a.amqpClient.Exclusive(context.Background(), a.amqpExclusiveRoutingKey, time.Hour, func(ctx context.Context) error {
		a.start(ctx, done)
		return nil
	}); err != nil {
		logger.Error("get exclusive semaphore: %s", err)
	}
}

func (a App) start(ctx context.Context, done <-chan struct{}) {
	logger.Info("Starting startup check...")
	defer logger.Info("Ending startup check.")

	err := a.storageApp.Walk(ctx, "", func(item absto.Item) error {
		select {
		case <-done:
			return errors.New("server is shutting down")
		default:
		}

		item = a.sanitizeName(ctx, item)
		a.notify(provider.NewStartEvent(item))

		return nil
	})
	if err != nil {
		logger.Error("%s", err)
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
