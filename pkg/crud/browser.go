package crud

import (
	"net/http"
	"path"

	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/httputils/v4/pkg/logger"
	"github.com/ViBiOh/httputils/v4/pkg/renderer"
	"github.com/ViBiOh/httputils/v4/pkg/sha"
)

// Browser render file web view
func (a App) Browser(w http.ResponseWriter, request provider.Request, item provider.StorageItem, message renderer.Message) (string, int, map[string]interface{}, error) {
	var (
		previous *provider.StorageItem
		next     *provider.StorageItem
	)

	pathParts := getPathParts(request.URL(""))
	breadcrumbs := pathParts[:len(pathParts)-1]

	files, err := a.storageApp.List(item.Dir())
	if err != nil {
		logger.WithField("item", item.Pathname).Error("unable to list neighbors files: %s", err)
	} else {
		previous, next = getPreviousAndNext(item, files)
	}

	return "file", http.StatusOK, map[string]interface{}{
		"Paths": breadcrumbs,
		"File": provider.RenderItem{
			ID:          sha.New(item.Name),
			StorageItem: item,
		},
		"Cover":     a.getCover(files),
		"HasStream": item.IsVideo() && a.thumbnailApp.HasStream(item),
		"Parent":    path.Join(breadcrumbs...),
		"Previous":  previous,
		"Next":      next,

		"Request": request,
		"Message": message,
	}, nil
}
