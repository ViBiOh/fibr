package crud

import (
	"net/http"

	absto "github.com/ViBiOh/absto/pkg/model"
	"github.com/ViBiOh/fibr/pkg/provider"
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

	thumbnails, err := a.rawStorageApp.List(ctx, a.thumbnailApp.Path(item))
	if err != nil {
		logger.WithField("item", item.Pathname).Error("unable to list thumbnail: %s", err)
	}

	items := make([]provider.StoryItem, 0, len(files))

	for _, item := range files {
		if !a.thumbnailApp.CanHaveThumbnail(item) {
			continue
		}

		thumbnailPath := a.thumbnailApp.PathForLarge(item)
		var found bool

		for _, thumbnailInfo := range thumbnails {
			if thumbnailInfo.Pathname != thumbnailPath {
				continue
			}

			found = true
			break
		}

		if !found {
			continue
		}

		exif, err := a.exifApp.GetExifFor(ctx, item)
		if err != nil {
			logger.WithField("item", item.Pathname).Error("unable to get exif: %s", err)
		}

		items = append(items, provider.StorageToStory(item, request, exif))
	}

	request.Display = provider.StoryDisplay

	return renderer.NewPage("story", http.StatusOK, map[string]interface{}{
		"Paths":              getPathParts(request),
		"Files":              items,
		"Cover":              a.getCover(ctx, request, files),
		"Request":            request,
		"ThumbnailLargeSize": a.thumbnailApp.LargeThumbnailSize(),
	}), nil
}
