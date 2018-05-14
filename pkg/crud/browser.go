package crud

import (
	"net/http"

	"github.com/ViBiOh/fibr/pkg/provider"
)

// Browser render file web view
func (a *App) Browser(w http.ResponseWriter, request *provider.Request, file *provider.StorageItem, message *provider.Message) {
	content := map[string]interface{}{
		`Paths`: getPathParts(request),
		`File`:  file,
	}

	a.renderer.File(w, request, content, message)
}
