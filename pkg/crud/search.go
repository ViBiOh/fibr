package crud

import (
	"net/http"

	absto "github.com/ViBiOh/absto/pkg/model"
	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/fibr/pkg/thumbnail"
	"github.com/ViBiOh/httputils/v4/pkg/renderer"
	"github.com/ViBiOh/httputils/v4/pkg/telemetry"
)

func (a App) search(r *http.Request, request provider.Request, item absto.Item, files []absto.Item) (renderer.Page, error) {
	ctx, end := telemetry.StartSpan(r.Context(), a.tracer, "search")
	defer end(nil)

	metadatas, err := a.metadataApp.GetAllMetadataFor(ctx, files...)
	if err != nil {
		listLogger(item.Pathname).Error("list metadatas: %s", err)
	}

	items := make([]provider.RenderItem, len(files))
	var hasMap bool

	renderWithThumbnail := request.Display == provider.GridDisplay

	for i, item := range files {
		renderItem := provider.StorageToRender(item, request)

		metadata := metadatas[item.ID]
		renderItem.Tags = metadata.Tags

		if renderWithThumbnail && a.thumbnailApp.CanHaveThumbnail(item) && a.thumbnailApp.HasThumbnail(ctx, item, thumbnail.SmallSize) {
			renderItem.HasThumbnail = true
		}

		items[i] = renderItem

		if !hasMap && metadata.Geocode.Longitude != 0 && metadata.Geocode.Latitude != 0 {
			hasMap = true
		}
	}

	return renderer.NewPage("search", http.StatusOK, map[string]any{
		"Paths":   getPathParts(request),
		"Files":   items,
		"Cover":   a.getCover(ctx, request, files),
		"Search":  r.URL.Query(),
		"Request": request,
		"HasMap":  hasMap,
	}), nil
}
