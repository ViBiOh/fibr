package crud

import (
	"context"
	"log/slog"
	"net/http"
	"sort"

	absto "github.com/ViBiOh/absto/pkg/model"
	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/fibr/pkg/thumbnail"
	"github.com/ViBiOh/httputils/v4/pkg/concurrent"
	"github.com/ViBiOh/httputils/v4/pkg/renderer"
	"github.com/ViBiOh/httputils/v4/pkg/telemetry"
	"go.opentelemetry.io/otel/trace"
)

func (a App) browse(ctx context.Context, request provider.Request, item absto.Item, message renderer.Message) (renderer.Page, error) {
	ctx, end := telemetry.StartSpan(ctx, a.tracer, "browse", trace.WithSpanKind(trace.SpanKindInternal))
	defer end(nil)

	var (
		previous provider.RenderItem
		next     provider.RenderItem
		files    []absto.Item
		metadata provider.Metadata
	)

	wg := concurrent.NewLimiter(-1)

	if request.Share.IsZero() || !request.Share.File {
		wg.Go(func() {
			files, previous, next = a.getFilesPreviousAndNext(ctx, item, request)
		})
	} else {
		files = []absto.Item{item}
	}

	wg.Go(func() {
		var err error
		metadata, err = a.metadataApp.GetMetadataFor(ctx, item)
		if err != nil && !absto.IsNotExist(err) {
			slog.Error("load metadata", "err", err, "item", item.Pathname)
		}
	})

	wg.Wait()

	renderItem := provider.StorageToRender(item, request)
	if a.thumbnailApp.CanHaveThumbnail(item) && a.thumbnailApp.HasThumbnail(ctx, item, thumbnail.SmallSize) {
		renderItem.HasThumbnail = true
	}

	return renderer.NewPage("file", http.StatusOK, map[string]any{
		"Paths":     getPathParts(request),
		"File":      renderItem,
		"Exif":      metadata,
		"Cover":     a.getCover(ctx, request, files),
		"HasStream": renderItem.IsVideo() && a.thumbnailApp.HasStream(ctx, item),

		"Previous": previous,
		"Next":     next,

		"Request": request,
		"Message": message,
	}), nil
}

func (a App) getFilesPreviousAndNext(ctx context.Context, item absto.Item, request provider.Request) (items []absto.Item, previous provider.RenderItem, next provider.RenderItem) {
	ctx, end := telemetry.StartSpan(ctx, a.tracer, "get_previous_next", trace.WithSpanKind(trace.SpanKindInternal))
	defer end(nil)

	var err error
	items, err = a.storageApp.List(ctx, item.Dir())
	if err != nil {
		slog.Error("list neighbors files", "err", err, "item", item.Pathname)
		return
	}

	sort.Sort(provider.ByHybridSort(items))

	previousItem, nextItem := getPreviousAndNext(item, items)

	if previousItem != nil {
		previous = provider.StorageToRender(*previousItem, request)
	}
	if nextItem != nil {
		next = provider.StorageToRender(*nextItem, request)
	}

	return
}
