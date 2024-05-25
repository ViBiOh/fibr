package crud

import (
	"log/slog"
	"net/http"

	absto "github.com/ViBiOh/absto/pkg/model"
	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/fibr/pkg/thumbnail"
	"github.com/ViBiOh/httputils/v4/pkg/concurrent"
	"github.com/ViBiOh/httputils/v4/pkg/renderer"
	"github.com/ViBiOh/httputils/v4/pkg/telemetry"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

func (s Service) story(r *http.Request, request provider.Request, item absto.Item, files []absto.Item) (renderer.Page, error) {
	ctx, end := telemetry.StartSpan(r.Context(), s.tracer, "story", trace.WithAttributes(attribute.String("item", item.Pathname)))
	defer end(nil)

	items := make([]provider.StoryItem, 0, len(files))
	var cover cover
	var hasMap bool

	wg := concurrent.NewLimiter(-1)

	var directoryAggregate provider.Aggregate
	wg.Go(func() {
		var err error

		directoryAggregate, err = s.metadata.GetAggregateFor(ctx, item)
		if err != nil && !absto.IsNotExist(err) {
			slog.LogAttrs(ctx, slog.LevelError, "get aggregate", slog.String("fn", "crud.story"), slog.String("item", request.Path), slog.Any("error", err))
		}
	})

	var exifs map[string]provider.Metadata
	wg.Go(func() {
		var err error

		exifs, err = s.metadata.GetAllMetadataFor(ctx, files...)
		if err != nil {
			slog.LogAttrs(ctx, slog.LevelError, "list exifs", slog.String("fn", "crud.story"), slog.String("item", request.Path), slog.Any("error", err))
		}
	})

	wg.Wait()

	for _, file := range files {
		if cover.IsZero() || (len(directoryAggregate.Cover) != 0 && cover.Img.Name() != directoryAggregate.Cover) {
			cover = newCover(provider.StorageToRender(file, request), thumbnail.SmallSize)
		}

		exif := exifs[file.ID]

		if !hasMap && exif.Coordinates != nil {
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
		"ThumbnailLargeSize": s.thumbnail.LargeThumbnailSize(),
	}), nil
}
