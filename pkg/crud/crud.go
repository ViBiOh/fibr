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
	// ErrNotAuthorized error returned when user is not authorized
	ErrNotAuthorized = errors.New("you're not authorized to do this ⛔")

	// ErrEmptyName error returned when user does not provide a name
	ErrEmptyName = errors.New("name is empty")

	// ErrEmptyFolder error returned when user does not provide a folder
	ErrEmptyFolder = errors.New("folder is empty")

	// ErrAbsoluteFolder error returned when user provide a relative folder
	ErrAbsoluteFolder = errors.New("folder has to be absolute")
)

// App of package
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

	rendererApp  renderer.App
	thumbnailApp thumbnail.App

	bcryptCost      int
	sanitizeOnStart bool
}

// Config of package
type Config struct {
	ignore                  *string
	amqpExclusiveRoutingKey *string
	bcryptDuration          *string
	sanitizeOnStart         *bool
}

// Flags adds flags for configuring package
func Flags(fs *flag.FlagSet, prefix string) Config {
	return Config{
		ignore:          flags.New(prefix, "crud", "IgnorePattern").Default("", nil).Label("Ignore pattern when listing files or directory").ToString(fs),
		sanitizeOnStart: flags.New(prefix, "crud", "SanitizeOnStart").Default(false, nil).Label("Sanitize name on start").ToBool(fs),
		bcryptDuration:  flags.New(prefix, "crud", "BcryptDuration").Default("0.25s", nil).Label("Wanted bcrypt duration for calculating effective cost").ToString(fs),

		amqpExclusiveRoutingKey: flags.New(prefix, "crud", "AmqpExclusiveRoutingKey").Default("fibr.semaphore.start", nil).Label("AMQP Routing Key for exclusive lock on default exchange").ToString(fs),
	}
}

// New creates new App from Config
func New(config Config, storage absto.Storage, rendererApp renderer.App, shareApp provider.ShareManager, webhookApp provider.WebhookManager, thumbnailApp thumbnail.App, exifApp exif.App, eventProducer provider.EventProducer, amqpClient *amqp.Client, tracerApp tracer.App) (App, error) {
	app := App{
		sanitizeOnStart: *config.sanitizeOnStart,

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
			return app, fmt.Errorf("unable to setup amqp exclusive: %s", err)
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
		if item.IsDir && item.Name == provider.MetadataDirectoryName {
			return true
		}

		if ignorePattern != nil && ignorePattern.MatchString(item.Name) {
			return true
		}

		return false
	})

	bcryptDuration, err := time.ParseDuration(strings.TrimSpace(*config.bcryptDuration))
	if err != nil {
		return app, fmt.Errorf("unable to parse bcrypt duration: %s", err)
	}

	bcryptCost, err := findBcryptBestCost(bcryptDuration)
	if err != nil {
		logger.Error("unable to find best bcrypt cost: %s", err)
		bcryptCost = bcrypt.DefaultCost
	}
	logger.Info("Best bcrypt cost is %d", bcryptCost)

	app.bcryptCost = bcryptCost

	return app, nil
}

// Start crud operations
func (a App) Start(done <-chan struct{}) {
	if a.amqpClient == nil {
		a.start(done)
		return
	}

	if _, err := a.amqpClient.Exclusive(context.Background(), a.amqpExclusiveRoutingKey, time.Hour, func(_ context.Context) error {
		a.start(done)
		return nil
	}); err != nil {
		logger.Error("unable to get exclusive semaphore: %s", err)
	}
}

func (a App) start(done <-chan struct{}) {
	logger.Info("Starting startup check...")
	defer logger.Info("Ending startup check.")

	err := a.storageApp.Walk("", func(item absto.Item) error {
		select {
		case <-done:
			return errors.New("server is shutting down")
		default:
		}

		item = a.sanitizeName(item)
		a.notify(provider.NewStartEvent(item))

		return nil
	})
	if err != nil {
		logger.Error("%s", err)
	}
}

func (a App) sanitizeName(item absto.Item) absto.Item {
	name, err := provider.SanitizeName(item.Pathname, false)
	if err != nil {
		logger.WithField("item", item.Pathname).Error("unable to sanitize name: %s", err)
		return item
	}

	if name == item.Pathname {
		return item
	}

	if !a.sanitizeOnStart {
		logger.Info("File with name `%s` should be renamed to `%s`", item.Pathname, name)
		return item
	}

	return a.rename(item, name)
}

func (a App) rename(item absto.Item, name string) absto.Item {
	logger.Info("Renaming `%s` to `%s`", item.Pathname, name)

	renamedItem, err := a.doRename(item.Pathname, name, item)
	if err != nil {
		logger.Error("%s", err)
		return item
	}

	return renamedItem
}

func (a App) error(w http.ResponseWriter, r *http.Request, request provider.Request, err error) {
	a.rendererApp.Error(w, r, map[string]interface{}{"Request": request}, err)
}
