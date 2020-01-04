package crud

import (
	"fmt"
	"net/http"
	"path"

	"github.com/ViBiOh/fibr/pkg/provider"
)

func getPreviousAndNext(file *provider.StorageItem, files []*provider.StorageItem) (previous *provider.StorageItem, next *provider.StorageItem) {
	found := false

	for _, neighbor := range files {
		if neighbor.IsDir != file.IsDir {
			continue
		}

		if neighbor.Name == file.Name {
			found = true
			continue
		}

		if !found {
			previous = neighbor
		}

		if found {
			next = neighbor
			return
		}
	}

	return
}

// Browser render file web view
func (a *app) Browser(w http.ResponseWriter, request *provider.Request, file *provider.StorageItem, message *provider.Message) {
	var previous, next *provider.StorageItem

	files, err := a.storage.List(path.Dir(file.Pathname))
	if err == nil {
		previous, next = getPreviousAndNext(file, files)
	}

	pathParts := getPathParts(request)

	content := map[string]interface{}{
		"Paths":    pathParts,
		"File":     file,
		"Parent":   fmt.Sprintf("/%s", path.Join(pathParts[:len(pathParts)-1]...)),
		"Previous": previous,
		"Next":     next,
	}

	a.renderer.File(w, request, content, message)
}
