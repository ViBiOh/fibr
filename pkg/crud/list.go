package crud

import (
	"net/http"
	"strings"

	"github.com/ViBiOh/fibr/pkg/provider"
)

// List render directory web view of given dirPath
func (a *App) List(w http.ResponseWriter, request *provider.Request, filename string, display string, message *provider.Message) {
	files, err := a.storage.List(filename)
	if err != nil {
		a.renderer.Error(w, http.StatusInternalServerError, err)
		return
	}

	paths := strings.Split(strings.Trim(request.Path, `/`), `/`)
	if len(paths) == 1 && paths[0] == `` {
		paths = nil
	}

	content := map[string]interface{}{
		`Paths`: paths,
		`Files`: files,
	}

	if request.CanShare {
		content[`Shares`] = a.metadatas
	}

	a.renderer.Directory(w, request, content, display, message)
}
