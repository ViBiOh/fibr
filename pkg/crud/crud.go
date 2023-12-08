package crud

import (
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
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

type Service struct {
	tracer          trace.Tracer
	rawStorage      absto.Storage
	storage         absto.Storage
	share           provider.ShareManager
	webhook         provider.WebhookManager
	metadata        provider.MetadataManager
	searchService   search.Service
	pushEvent       provider.EventProducer
	temporaryFolder string
	renderer        *renderer.Service
	thumbnail       thumbnail.Service
	bcryptCost      int
	chunkUpload     bool
}

type Config struct {
	BcryptDuration  string
	TemporaryFolder string
	ChunkUpload     bool
}

func Flags(fs *flag.FlagSet, prefix string) *Config {
	var config Config

	flags.New("BcryptDuration", "Wanted bcrypt duration for calculating effective cost").Prefix(prefix).DocPrefix("crud").StringVar(fs, &config.BcryptDuration, "0.25s", nil)
	flags.New("ChunkUpload", "Use chunk upload in browser").Prefix(prefix).DocPrefix("crud").BoolVar(fs, &config.ChunkUpload, false, nil)
	flags.New("TemporaryFolder", "Temporary folder for chunk upload").Prefix(prefix).DocPrefix("crud").StringVar(fs, &config.TemporaryFolder, "/tmp", nil)

	return &config
}

func New(config *Config, storageService absto.Storage, filteredStorage absto.Storage, rendererService *renderer.Service, shareService provider.ShareManager, webhookService provider.WebhookManager, thumbnailService thumbnail.Service, exifService provider.MetadataManager, searchService search.Service, eventProducer provider.EventProducer, tracerProvider trace.TracerProvider) (Service, error) {
	service := Service{
		chunkUpload:     config.ChunkUpload,
		temporaryFolder: config.TemporaryFolder,
		pushEvent:       eventProducer,
		rawStorage:      storageService,
		storage:         filteredStorage,
		renderer:        rendererService,
		thumbnail:       thumbnailService,
		metadata:        exifService,
		share:           shareService,
		webhook:         webhookService,
		searchService:   searchService,
	}

	if tracerProvider != nil {
		service.tracer = tracerProvider.Tracer("crud")
	}

	bcryptDuration, err := time.ParseDuration(config.BcryptDuration)
	if err != nil {
		return service, fmt.Errorf("parse bcrypt duration: %w", err)
	}

	bcryptCost, err := bcrypt.FindBestCost(bcryptDuration)
	if err != nil {
		slog.Error("find best bcrypt cost", "error", err)
	}
	slog.Info("Best bcrypt cost computed", "cost", bcryptCost)

	service.bcryptCost = bcryptCost

	return service, nil
}

func (s Service) error(w http.ResponseWriter, r *http.Request, request provider.Request, err error) {
	s.renderer.Error(w, r, map[string]any{"Request": request}, err)
}
