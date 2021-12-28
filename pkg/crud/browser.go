package crud

import (
	"net/http"
	"path"

	exas "github.com/ViBiOh/exas/pkg/model"
	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/httputils/v4/pkg/concurrent"
	"github.com/ViBiOh/httputils/v4/pkg/logger"
	"github.com/ViBiOh/httputils/v4/pkg/renderer"
	"github.com/ViBiOh/httputils/v4/pkg/sha"
)

// Browser render file web view
func (a App) Browser(w http.ResponseWriter, request provider.Request, item provider.StorageItem, message renderer.Message) (string, int, map[string]interface{}, error) {
	var (
		previous *provider.StorageItem
		next     *provider.StorageItem
		files    []provider.StorageItem
		exif     exas.Exif
	)

	pathParts := getPathParts(request.SelfURL)
	breadcrumbs := pathParts[:len(pathParts)-1]

	wg := concurrent.NewSimple()

	wg.Go(func() {
		var err error
		files, err = a.storageApp.List(item.Dir())
		if err != nil {
			logger.WithField("item", item.Pathname).Error("unable to list neighbors files: %s", err)
		} else {
			previous, next = getPreviousAndNext(item, files)
		}
	})

	wg.Go(func() {
		var err error
		exif, err = a.exifApp.GetExifFor(item)
		if err != nil {
			logger.WithField("item", item.Pathname).Error("unable to load exif: %s", err)
		}
	})

	wg.Wait()

	return "file", http.StatusOK, map[string]interface{}{
		"Paths": breadcrumbs,
		"File": provider.RenderItem{
			ID:          sha.New(item.Name),
			StorageItem: item,
		},
		"Exif":      exif,
		"Cover":     a.getCover(files),
		"HasStream": item.IsVideo() && a.thumbnailApp.HasStream(item),
		"Parent":    path.Join(breadcrumbs...),
		"Previous":  previous,
		"Next":      next,

		"Request": request,
		"Message": message,
	}, nil
}
