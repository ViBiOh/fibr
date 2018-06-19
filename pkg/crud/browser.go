package crud

import (
	"net/http"
	"path"

	"github.com/ViBiOh/fibr/pkg/provider"
)

func getPreviousAndNext(file *provider.StorageItem, files []*provider.StorageItem) (previous *provider.StorageItem, next *provider.StorageItem) {
	found := false
	for _, neighbor := range files {
		if neighbor.Pathname == file.Pathname {
			found = true
			continue
		}

		if !found && !neighbor.IsDir {
			previous = neighbor
		}

		if found && !neighbor.IsDir {
			next = neighbor
			return
		}
	}

	return
}

// Browser render file web view
func (a *App) Browser(w http.ResponseWriter, request *provider.Request, file *provider.StorageItem, message *provider.Message) {
	var previous, next *provider.StorageItem

	files, err := a.storage.List(path.Dir(file.Pathname))
	if err == nil {
		previous, next = getPreviousAndNext(file, files)
	}

	content := map[string]interface{}{
		`Paths`:    getPathParts(request),
		`File`:     file,
		`Previous`: previous,
		`Next`:     next,
	}

	a.renderer.File(w, request, content, message)
}
