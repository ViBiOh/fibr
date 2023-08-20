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

type App struct {
	tracer       trace.Tracer
	storageApp   absto.Storage
	exifApp      provider.MetadataManager
	exclusiveApp exclusive.App
	thumbnailApp thumbnail.App
}

func New(storageApp absto.Storage, thumbnailApp thumbnail.App, exifApp provider.MetadataManager, exclusiveApp exclusive.App, tracerProvider trace.TracerProvider) App {
	app := App{
		storageApp:   storageApp,
		thumbnailApp: thumbnailApp,
		exifApp:      exifApp,
		exclusiveApp: exclusiveApp,
	}

	if tracerProvider != nil {
		app.tracer = tracerProvider.Tracer("search")
	}

	return app
}

func (a App) Files(r *http.Request, request provider.Request) (items []absto.Item, err error) {
	params := r.URL.Query()

	ctx, end := telemetry.StartSpan(r.Context(), a.tracer, "filter")
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
