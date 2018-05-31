package crud

import (
	"net/http"
	"path"

	"github.com/ViBiOh/fibr/pkg/provider"
)

// Browser render file web view
func (a *App) Browser(w http.ResponseWriter, request *provider.Request, file *provider.StorageItem, message *provider.Message) {
	var previous, next *provider.StorageItem

	files, err := a.storage.List(path.Dir(file.Pathname))
	if err == nil {
		for i, neighbor := range files {
			if neighbor.Pathname == file.Pathname {
				if i > 0 {
					previous = files[i-1]
				}

				if i != len(files)-1 {
					next = files[i+1]
				}

				break
			}
		}
	}

	content := map[string]interface{}{
		`Paths`:    getPathParts(request),
		`File`:     file,
		`Previous`: previous,
		`Next`:     next,
	}

	a.renderer.File(w, request, content, message)
}
