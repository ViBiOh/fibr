package search

import (
	"log/slog"
	"net/http"
	"time"

	absto "github.com/ViBiOh/absto/pkg/model"
	"github.com/ViBiOh/fibr/pkg/exclusive"
	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/fibr/pkg/thumbnail"
	httpModel "github.com/ViBiOh/httputils/v4/pkg/model"
	"github.com/ViBiOh/httputils/v4/pkg/telemetry"
	"go.opentelemetry.io/otel/trace"
)

type Service struct {
	tracer    trace.Tracer
	storage   absto.Storage
	exif      provider.MetadataManager
	exclusive exclusive.Service
	thumbnail thumbnail.Service
}

func New(storageService absto.Storage, thumbnailService thumbnail.Service, exifService provider.MetadataManager, exclusiveService exclusive.Service, tracerProvider trace.TracerProvider) Service {
	service := Service{
		storage:   storageService,
		thumbnail: thumbnailService,
		exif:      exifService,
		exclusive: exclusiveService,
	}

	if tracerProvider != nil {
		service.tracer = tracerProvider.Tracer("search")
	}

	return service
}

func (s Service) Files(r *http.Request, request provider.Request) (items []absto.Item, err error) {
	params := r.URL.Query()

	ctx, end := telemetry.StartSpan(r.Context(), s.tracer, "filter")
	defer end(&err)

	criterions, err := parseSearch(params, time.Now())
	if err != nil {
		return nil, httpModel.WrapInvalid(err)
	}

	hasTags := criterions.hasTags()

	err = s.storage.Walk(ctx, request.Filepath(), func(item absto.Item) error {
		if item.IsDir() || !criterions.match(item) {
			return nil
		}

		if hasTags {
			metadata, err := s.exif.GetMetadataFor(ctx, item)
			if err != nil {
				slog.Error("get metadata", "err", err, "item", item.Pathname)
			}

			if !criterions.matchTags(metadata) {
				return nil
			}
		}

		items = append(items, item)

		return nil
	})

	return
}
