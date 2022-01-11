package crud

import (
	"net/http"

	absto "github.com/ViBiOh/absto/pkg/model"
	exas "github.com/ViBiOh/exas/pkg/model"
	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/httputils/v4/pkg/concurrent"
	"github.com/ViBiOh/httputils/v4/pkg/logger"
	"github.com/ViBiOh/httputils/v4/pkg/renderer"
)

// Browser render file web view
func (a App) Browser(w http.ResponseWriter, request provider.Request, item absto.Item, message renderer.Message) (string, int, map[string]interface{}, error) {
	var (
		previous *provider.RenderItem
		next     *provider.RenderItem
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
		exif, err = a.exifApp.GetExifFor(item)
		if err != nil {
			logger.WithField("item", item.Pathname).Error("unable to load exif: %s", err)
		}
	})

	wg.Wait()

	return "file", http.StatusOK, map[string]interface{}{
		"Paths":     getPathParts(request),
		"File":      provider.StorageToRender(item, request),
		"Exif":      exif,
		"Cover":     a.getCover(files),
		"HasStream": item.IsVideo() && a.thumbnailApp.HasStream(item),

		"Previous": previous,
		"Next":     next,

		"Request": request,
		"Message": message,
	}, nil
}

func (a App) getFilesPreviousAndNext(item absto.Item, request provider.Request) (files []absto.Item, previous *provider.RenderItem, next *provider.RenderItem) {
	var err error
	files, err = a.storageApp.List(item.Dir())
	if err != nil {
		logger.WithField("item", item.Pathname).Error("unable to list neighbors files: %s", err)
		return
	}

	previousItem, nextItem := getPreviousAndNext(item, files)

	if previousItem != nil {
		content := provider.StorageToRender(*previousItem, request)
		previous = &content
	}
	if nextItem != nil {
		content := provider.StorageToRender(*nextItem, request)
		next = &content
	}

	return
}
