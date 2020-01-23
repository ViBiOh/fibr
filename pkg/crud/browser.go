package crud

import (
	"net/http"
	"path"

	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/httputils/v3/pkg/logger"
)

// Browser render file web view
func (a *app) Browser(w http.ResponseWriter, request provider.Request, file provider.StorageItem, message *provider.Message) {
	var (
		previous *provider.StorageItem
		next     *provider.StorageItem
	)

	pathParts := getPathParts(request.GetURI(""))
	breadcrumbs := pathParts[:len(pathParts)-1]

	files, err := a.storage.List(path.Dir(file.Pathname))
	if err != nil {
		logger.Error("unable to list neighbors files: %s", err)
	} else {
		previous, next = getPreviousAndNext(file, files)
	}

	content := map[string]interface{}{
		"Paths":    breadcrumbs,
		"File":     file,
		"Cover":    a.getCover(files),
		"Parent":   path.Join(breadcrumbs...),
		"Previous": previous,
		"Next":     next,
	}

	a.renderer.File(w, request, content, message)
}
