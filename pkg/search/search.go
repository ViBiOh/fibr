package search

import (
	"net/http"
	"time"

	absto "github.com/ViBiOh/absto/pkg/model"
	"github.com/ViBiOh/fibr/pkg/exclusive"
	"github.com/ViBiOh/fibr/pkg/exif"
	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/fibr/pkg/thumbnail"
	httpModel "github.com/ViBiOh/httputils/v4/pkg/model"
	"github.com/ViBiOh/httputils/v4/pkg/tracer"
	"go.opentelemetry.io/otel/trace"
)

type App struct {
	tracer       trace.Tracer
	storageApp   absto.Storage
	exifApp      provider.ExifManager
	exclusiveApp exclusive.App
	thumbnailApp thumbnail.App
}

func New(storageApp absto.Storage, thumbnailApp thumbnail.App, exifApp exif.App, exclusiveApp exclusive.App, tracer trace.Tracer) App {
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

	criterions, err := parseSearch(params, time.Now())
	if err != nil {
		return nil, httpModel.WrapInvalid(err)
	}

	err = a.storageApp.Walk(r.Context(), request.Filepath(), func(item absto.Item) error {
		if item.IsDir || !criterions.match(item) {
			return nil
		}

		items = append(items, item)

		return nil
	})

	return
}

func (a App) Search(r *http.Request, request provider.Request, files []absto.Item) ([]provider.RenderItem, bool, error) {
	ctx, end := tracer.StartSpan(r.Context(), a.tracer, "search")
	defer end()

	items := make([]provider.RenderItem, len(files))
	var hasMap bool

	renderWithThumbnail := request.Display == provider.GridDisplay

	for i, item := range files {
		renderItem := provider.StorageToRender(item, request)

		if renderWithThumbnail && a.thumbnailApp.CanHaveThumbnail(item) && a.thumbnailApp.HasThumbnail(ctx, item, thumbnail.SmallSize) {
			renderItem.HasThumbnail = true
		}

		items[i] = renderItem

		if !hasMap {
			if exif, err := a.exifApp.GetMetadataFor(ctx, item); err == nil && exif.Geocode.Longitude != 0 && exif.Geocode.Latitude != 0 {
				hasMap = true
			}
		}
	}

	return items, hasMap, nil
}
