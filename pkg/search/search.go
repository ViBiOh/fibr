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

	err = s.storage.Walk(ctx, request.Filepath(), func(item absto.Item) error {
		if !item.IsDir() && criterions.match(item) {
			items = append(items, item)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	if criterions.hasTags() {
		metadatas, metaErr := s.exif.GetAllMetadataFor(ctx, items...)
		if metaErr != nil {
			slog.LogAttrs(ctx, slog.LevelError, "get all metadata", slog.Any("error", metaErr))
		}

		filtered := items[:0]

		for _, item := range items {
			if criterions.matchTags(metadatas[item.ID]) {
				filtered = append(filtered, item)
			}
		}

		items = filtered
	}

	return items, err
}
