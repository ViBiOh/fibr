package crud

import (
	"net/http"

	absto "github.com/ViBiOh/absto/pkg/model"
	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/fibr/pkg/thumbnail"
	"github.com/ViBiOh/httputils/v4/pkg/logger"
	"github.com/ViBiOh/httputils/v4/pkg/renderer"
	"go.opentelemetry.io/otel/trace"
)

func (a App) story(r *http.Request, request provider.Request, item absto.Item, files []absto.Item) (renderer.Page, error) {
	ctx := r.Context()
	if a.tracer != nil {
		var span trace.Span
		ctx, span = a.tracer.Start(ctx, "story")
		defer span.End()
	}

	thumbnails, err := a.thumbnailApp.ListDirLarge(ctx, item)
	if err != nil {
		logger.WithField("item", item.Pathname).Error("unable to list thumbnail: %s", err)
	}

	items := make([]provider.StoryItem, 0, len(files))
	var cover map[string]any

	for _, item := range files {
		if _, ok := thumbnails[a.thumbnailApp.PathForLarge(item)]; !ok {
			continue
		}

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

		items = append(items, provider.StorageToStory(item, request, exif))
	}

	request.Display = provider.StoryDisplay

	return renderer.NewPage("story", http.StatusOK, map[string]any{
		"Paths":              getPathParts(request),
		"Files":              items,
		"Cover":              cover,
		"Request":            request,
		"ThumbnailLargeSize": a.thumbnailApp.LargeThumbnailSize(),
	}), nil
}
