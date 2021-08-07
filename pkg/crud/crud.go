package crud

import (
	"errors"
	"flag"
	"regexp"
	"strings"
	"time"

	"github.com/ViBiOh/fibr/pkg/exif"
	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/fibr/pkg/thumbnail"
	"github.com/ViBiOh/httputils/v4/pkg/flags"
	"github.com/ViBiOh/httputils/v4/pkg/logger"
	"github.com/ViBiOh/httputils/v4/pkg/renderer"
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
	storageApp   provider.Storage
	rendererApp  renderer.App
	shareApp     provider.ShareManager
	thumbnailApp thumbnail.App
	exifApp      exif.App

	pushEvent provider.EventProducer

	bcryptCost      int
	sanitizeOnStart bool
}

// Config of package
type Config struct {
	ignore          *string
	sanitizeOnStart *bool
}

// Flags adds flags for configuring package
func Flags(fs *flag.FlagSet, prefix string) Config {
	return Config{
		ignore:          flags.New(prefix, "crud").Name("IgnorePattern").Default("").Label("Ignore pattern when listing files or directory").ToString(fs),
		sanitizeOnStart: flags.New(prefix, "crud").Name("SanitizeOnStart").Default(false).Label("Sanitize name on start").ToBool(fs),
	}
}

// New creates new App from Config
func New(config Config, storage provider.Storage, rendererApp renderer.App, shareApp provider.ShareManager, thumbnailApp thumbnail.App, exifApp exif.App, eventProducer provider.EventProducer) (App, error) {
	app := App{
		sanitizeOnStart: *config.sanitizeOnStart,

		pushEvent: eventProducer,

		storageApp:   storage,
		rendererApp:  rendererApp,
		thumbnailApp: thumbnailApp,
		exifApp:      exifApp,
		shareApp:     shareApp,
	}

	var ignorePattern *regexp.Regexp
	ignore := strings.TrimSpace(*config.ignore)
	if len(ignore) != 0 {
		pattern, err := regexp.Compile(ignore)
		if err != nil {
			return App{}, err
		}

		ignorePattern = pattern
		logger.Info("Ignoring files with pattern `%s`", ignore)
	}

	storage.SetIgnoreFn(func(item provider.StorageItem) bool {
		if item.IsDir && item.Name == provider.MetadataDirectoryName {
			return true
		}

		if ignorePattern != nil && ignorePattern.MatchString(item.Name) {
			return true
		}

		return false
	})

	bcryptCost, err := findBcryptBestCost(time.Second / 4)
	if err != nil {
		logger.Error("unable to find best bcrypt cost: %s", err)
		bcryptCost = bcrypt.DefaultCost
	}
	logger.Info("Best bcrypt cost is %d", bcryptCost)

	app.bcryptCost = bcryptCost

	return app, nil
}

// Start crud operations
func (a *App) Start(done <-chan struct{}) {
	err := a.storageApp.Walk("", func(item provider.StorageItem, err error) error {
		if err != nil {
			return err
		}

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

func (a *App) sanitizeName(item provider.StorageItem) provider.StorageItem {
	name, err := provider.SanitizeName(item.Pathname, false)
	if err != nil {
		logger.Error("unable to sanitize name %s: %s", item.Pathname, err)
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

func (a *App) rename(item provider.StorageItem, name string) provider.StorageItem {
	logger.Info("Renaming `%s` to `%s`", item.Pathname, name)

	renamedItem, err := a.doRename(item.Pathname, name, item)
	if err != nil {
		logger.Error("%s", err)
		return item
	}

	return renamedItem
}
