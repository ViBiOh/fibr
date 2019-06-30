package crud

import (
	"net/http"

	"github.com/ViBiOh/fibr/pkg/provider"
)

// List render directory web view of given dirPath
func (a *app) List(w http.ResponseWriter, request *provider.Request, message *provider.Message) {
	files, err := a.storage.List(request.GetFilepath(""))
	if err != nil {
		a.renderer.Error(w, provider.NewError(http.StatusInternalServerError, err))
		return
	}

	content := map[string]interface{}{
		"Paths": getPathParts(request),
		"Files": files,
	}

	if request.CanShare {
		content["Shares"] = a.metadatas
	}

	a.renderer.Directory(w, request, content, message)
}
