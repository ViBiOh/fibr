package crud

import (
	"net/http"

	absto "github.com/ViBiOh/absto/pkg/model"
	exas "github.com/ViBiOh/exas/pkg/model"
	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/fibr/pkg/thumbnail"
	"github.com/ViBiOh/httputils/v4/pkg/concurrent"
	"github.com/ViBiOh/httputils/v4/pkg/logger"
	"github.com/ViBiOh/httputils/v4/pkg/renderer"
	"github.com/ViBiOh/httputils/v4/pkg/tracer"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

func (a App) story(r *http.Request, request provider.Request, item absto.Item, files []absto.Item) (renderer.Page, error) {
	ctx, end := tracer.StartSpan(r.Context(), a.tracer, "story", trace.WithAttributes(attribute.String("item", item.Pathname)))
	defer end()

	items := make([]provider.StoryItem, 0, len(files))
	var cover cover
	var hasMap bool

	wg := concurrent.NewSimple()

	var directoryAggregate provider.Aggregate
	wg.Go(func() {
		var err error

		directoryAggregate, err = a.exifApp.GetAggregateFor(ctx, item)
		if err != nil && !absto.IsNotExist(err) {
			logger.WithField("fn", "crud.story").WithField("item", request.Path).Error("get aggregate: %s", err)
		}
	})

	var exifs map[string]exas.Exif
	wg.Go(func() {
		var err error

		exifs, err = a.exifApp.ListExifFor(ctx, files...)
		if err != nil {
			logger.WithField("fn", "crud.story").WithField("item", request.Path).Error("list exifs: %s", err)
		}
	})

	wg.Wait()

	for _, file := range files {
		if cover.IsZero() || (len(directoryAggregate.Cover) != 0 && cover.Img.Name != directoryAggregate.Cover) {
			cover = newCover(provider.StorageToRender(file, request), thumbnail.SmallSize)
		}

		exif := exifs[file.ID]

		if !request.Share.Story && !hasMap && exif.Geocode.HasCoordinates() {
			hasMap = true
		}

		items = append(items, provider.StorageToStory(file, request, exif))
	}

	return renderer.NewPage("story", http.StatusOK, map[string]any{
		"Paths":              getPathParts(request),
		"Files":              items,
		"Cover":              cover,
		"Request":            request,
		"HasMap":             hasMap,
		"ThumbnailLargeSize": a.thumbnailApp.LargeThumbnailSize(),
	}), nil
}
