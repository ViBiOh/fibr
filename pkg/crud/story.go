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
	var cover map[string]any
	var hasMap bool

	for _, item := range files {
		if cover == nil {
			cover = map[string]any{
				"Img":       provider.StorageToRender(item, request),
				"ImgHeight": thumbnail.SmallSize,
				"ImgWidth":  thumbnail.SmallSize,
			}
		}

		exif, err := a.exifApp.GetExifFor(ctx, item)
		if err != nil {
			logger.WithField("item", item.Pathname).Error("unable to get exif: %s", err)
		}

		if !request.Share.Story && !hasMap && exif.Geocode.HasCoordinates() {
			hasMap = true
		}

		items = append(items, provider.StorageToStory(item, request, exif))
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
