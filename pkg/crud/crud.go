package crud

import (
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	absto "github.com/ViBiOh/absto/pkg/model"
	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/fibr/pkg/search"
	"github.com/ViBiOh/fibr/pkg/thumbnail"
	"github.com/ViBiOh/flags"
	"github.com/ViBiOh/httputils/v4/pkg/bcrypt"
	"github.com/ViBiOh/httputils/v4/pkg/renderer"
	"go.opentelemetry.io/otel/trace"
)

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
	temporaryFolder string
	rendererApp     *renderer.App
	thumbnailApp    thumbnail.App
	bcryptCost      int
	chunkUpload     bool
}

type Config struct {
	bcryptDuration  *string
	temporaryFolder *string
	chunkUpload     *bool
}

func Flags(fs *flag.FlagSet, prefix string) Config {
	return Config{
		bcryptDuration:  flags.New("BcryptDuration", "Wanted bcrypt duration for calculating effective cost").Prefix(prefix).DocPrefix("crud").String(fs, "0.25s", nil),
		chunkUpload:     flags.New("ChunkUpload", "Use chunk upload in browser").Prefix(prefix).DocPrefix("crud").Bool(fs, false, nil),
		temporaryFolder: flags.New("TemporaryFolder", "Temporary folder for chunk upload").Prefix(prefix).DocPrefix("crud").String(fs, "/tmp", nil),
	}
}

func New(config Config, storageApp absto.Storage, filteredStorage absto.Storage, rendererApp *renderer.App, shareApp provider.ShareManager, webhookApp provider.WebhookManager, thumbnailApp thumbnail.App, exifApp provider.MetadataManager, searchApp search.App, eventProducer provider.EventProducer, tracer trace.Tracer) (App, error) {
	app := App{
		chunkUpload:     *config.chunkUpload,
		temporaryFolder: strings.TrimSpace(*config.temporaryFolder),
		tracer:          tracer,
		pushEvent:       eventProducer,
		rawStorageApp:   storageApp,
		storageApp:      filteredStorage,
		rendererApp:     rendererApp,
		thumbnailApp:    thumbnailApp,
		metadataApp:     exifApp,
		shareApp:        shareApp,
		webhookApp:      webhookApp,
		searchApp:       searchApp,
	}

	bcryptDuration, err := time.ParseDuration(strings.TrimSpace(*config.bcryptDuration))
	if err != nil {
		return app, fmt.Errorf("parse bcrypt duration: %w", err)
	}

	bcryptCost, err := bcrypt.FindBestCost(bcryptDuration)
	if err != nil {
		slog.Error("find best bcrypt cost", "err", err)
	}
	slog.Info("Best bcrypt cost computed", "cost", bcryptCost)

	app.bcryptCost = bcryptCost

	return app, nil
}

func (a App) error(w http.ResponseWriter, r *http.Request, request provider.Request, err error) {
	a.rendererApp.Error(w, r, map[string]any{"Request": request}, err)
}
