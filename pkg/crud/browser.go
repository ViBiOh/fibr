package crud

import (
	"fmt"
	"net/http"
	"path"

	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/httputils/v3/pkg/logger"
)

// Browser render file web view
func (a *app) Browser(w http.ResponseWriter, request *provider.Request, file *provider.StorageItem, message *provider.Message) {
	var (
		previous *provider.StorageItem
		next     *provider.StorageItem
	)
	pathParts := getPathParts(request)

	if files, err := a.storage.List(path.Dir(file.Pathname)); err != nil {
		logger.Error("unable to list neighbors files: %s", err)
	} else {
		previous, next = getPreviousAndNext(file, files)
	}

	content := map[string]interface{}{
		"Paths":    pathParts,
		"File":     file,
		"Parent":   fmt.Sprintf("/%s", path.Join(pathParts[:len(pathParts)-1]...)),
		"Previous": previous,
		"Next":     next,
	}

	a.renderer.File(w, request, content, message)
}
