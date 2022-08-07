package crud

import (
	"net/http"

	absto "github.com/ViBiOh/absto/pkg/model"
	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/fibr/pkg/thumbnail"
	"github.com/ViBiOh/httputils/v4/pkg/logger"
	"github.com/ViBiOh/httputils/v4/pkg/renderer"
	"github.com/ViBiOh/httputils/v4/pkg/tracer"
)

func (a App) story(r *http.Request, request provider.Request, item absto.Item, files []absto.Item) (renderer.Page, error) {
	ctx, end := tracer.StartSpan(r.Context(), a.tracer, "story")
	defer end()

	items := make([]provider.StoryItem, 0, len(files))
	var cover cover
	var hasMap bool

	directoryAggregate, err := a.exifApp.GetAggregateFor(ctx, item)
	if err != nil && !absto.IsNotExist(err) {
		logger.WithField("fn", "crud.story").WithField("item", request.Path).Error("get aggregate: %s", err)
	}

	for _, file := range files {
		if cover.IsZero() || (len(directoryAggregate.Cover) != 0 && cover.Img.Name != directoryAggregate.Cover) {
			cover = newCover(provider.StorageToRender(file, request), thumbnail.SmallSize)
		}

		exif, err := a.exifApp.GetExifFor(ctx, file)
		if err != nil {
			logger.WithField("item", file.Pathname).Error("get exif: %s", err)
		}

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
