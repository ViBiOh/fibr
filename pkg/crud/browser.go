package crud

import (
	"context"
	"net/http"
	"sort"

	absto "github.com/ViBiOh/absto/pkg/model"
	exas "github.com/ViBiOh/exas/pkg/model"
	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/fibr/pkg/thumbnail"
	"github.com/ViBiOh/httputils/v4/pkg/concurrent"
	"github.com/ViBiOh/httputils/v4/pkg/logger"
	"github.com/ViBiOh/httputils/v4/pkg/renderer"
)

// Browser render file web view
func (a App) Browser(ctx context.Context, w http.ResponseWriter, request provider.Request, item absto.Item, message renderer.Message) (renderer.Page, error) {
	var (
		previous provider.RenderItem
		next     provider.RenderItem
		files    []absto.Item
		exif     exas.Exif
	)

	wg := concurrent.NewSimple()

	if request.Share.IsZero() || !request.Share.File {
		wg.Go(func() {
			files, previous, next = a.getFilesPreviousAndNext(item, request)
		})
	} else {
		files = []absto.Item{item}
	}

	wg.Go(func() {
		var err error
		exif, err = a.exifApp.GetExifFor(ctx, item)
		if err != nil {
			logger.WithField("item", item.Pathname).Error("unable to load exif: %s", err)
		}
	})

	wg.Wait()

	renderItem := provider.StorageToRender(item, request)
	if a.thumbnailApp.CanHaveThumbnail(item) && a.thumbnailApp.HasThumbnail(item, thumbnail.SmallSize) {
		renderItem.HasThumbnail = true
	}

	return renderer.NewPage("file", http.StatusOK, map[string]interface{}{
		"Paths":     getPathParts(request),
		"File":      renderItem,
		"Exif":      exif,
		"Cover":     a.getCover(request, files),
		"HasStream": renderItem.IsVideo() && a.thumbnailApp.HasStream(item),

		"Previous": previous,
		"Next":     next,

		"Request": request,
		"Message": message,
	}), nil
}

func (a App) getFilesPreviousAndNext(item absto.Item, request provider.Request) (items []absto.Item, previous provider.RenderItem, next provider.RenderItem) {
	var err error
	items, err = a.storageApp.List(item.Dir())
	if err != nil {
		logger.WithField("item", item.Pathname).Error("unable to list neighbors files: %s", err)
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
