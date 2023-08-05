package search

import (
	"net/http"
	"time"

	absto "github.com/ViBiOh/absto/pkg/model"
	"github.com/ViBiOh/fibr/pkg/exclusive"
	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/fibr/pkg/thumbnail"
	"github.com/ViBiOh/httputils/v4/pkg/logger"
	httpModel "github.com/ViBiOh/httputils/v4/pkg/model"
	"github.com/ViBiOh/httputils/v4/pkg/tracer"
	"go.opentelemetry.io/otel/trace"
)

type App struct {
	tracer       trace.Tracer
	storageApp   absto.Storage
	exifApp      provider.MetadataManager
	exclusiveApp exclusive.App
	thumbnailApp thumbnail.App
}

func New(storageApp absto.Storage, thumbnailApp thumbnail.App, exifApp provider.MetadataManager, exclusiveApp exclusive.App, tracer trace.Tracer) App {
	return App{
		tracer:       tracer,
		storageApp:   storageApp,
		thumbnailApp: thumbnailApp,
		exifApp:      exifApp,
		exclusiveApp: exclusiveApp,
	}
}

func (a App) Files(r *http.Request, request provider.Request) (items []absto.Item, err error) {
	params := r.URL.Query()

	ctx, end := tracer.StartSpan(r.Context(), a.tracer, "filter")
	defer end(&err)

	criterions, err := parseSearch(params, time.Now())
	if err != nil {
		return nil, httpModel.WrapInvalid(err)
	}

	hasTags := criterions.hasTags()

	err = a.storageApp.Walk(ctx, request.Filepath(), func(item absto.Item) error {
		if item.IsDir() || !criterions.match(item) {
			return nil
		}

		if hasTags {
			metadata, err := a.exifApp.GetMetadataFor(ctx, item)
			if err != nil {
				logger.WithField("item", item.Pathname).Error("get metadata: %s", err)
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
