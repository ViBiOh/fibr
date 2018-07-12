package crud

import (
	"net/http"

	"github.com/ViBiOh/fibr/pkg/provider"
)

// List render directory web view of given dirPath
func (a *App) List(w http.ResponseWriter, request *provider.Request, display string, message *provider.Message) {
	files, err := a.storage.List(request.GetPath())
	if err != nil {
		a.renderer.Error(w, http.StatusInternalServerError, err)
		return
	}

	content := map[string]interface{}{
		`Paths`: getPathParts(request),
		`Files`: files,
	}

	if request.CanShare {
		content[`Shares`] = a.metadatas
	}

	a.renderer.Directory(w, request, content, display, message)
}
