package crud

import (
	"net/http"
	"path"

	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/fibr/pkg/sha"
	"github.com/ViBiOh/httputils/v4/pkg/logger"
	"github.com/ViBiOh/httputils/v4/pkg/renderer"
)

// Browser render file web view
func (a *App) Browser(w http.ResponseWriter, request provider.Request, file provider.StorageItem, message renderer.Message) (string, int, map[string]interface{}, error) {
	var (
		previous *provider.StorageItem
		next     *provider.StorageItem
	)

	pathParts := getPathParts(request.URL(""))
	breadcrumbs := pathParts[:len(pathParts)-1]

	files, err := a.storageApp.List(file.Dir())
	if err != nil {
		logger.Error("unable to list neighbors files: %s", err)
	} else {
		previous, next = getPreviousAndNext(file, files)
	}

	return "file", http.StatusOK, map[string]interface{}{
		"Paths": breadcrumbs,
		"File": provider.RenderItem{
			ID:          sha.Sha1(file.Name),
			StorageItem: file,
		},
		"Cover":    a.getCover(files),
		"Parent":   path.Join(breadcrumbs...),
		"Previous": previous,
		"Next":     next,

		"Request": request,
		"Message": message,
	}, nil
}
