package crud

import (
	"fmt"
	"net/http"

	"github.com/ViBiOh/fibr/pkg/provider"
)

// Delete given path from filesystem
func (a *app) Delete(w http.ResponseWriter, r *http.Request, request *provider.Request) {
	if !request.CanEdit {
		a.renderer.Error(w, http.StatusForbidden, ErrNotAuthorized)
		return
	}

	pathname, err := getFilepath(r, request)
	if err != nil && err == ErrNotAuthorized {
		a.renderer.Error(w, http.StatusForbidden, err)
		return
	}

	info, err := a.storage.Info(pathname)
	if err != nil {
		a.renderer.Error(w, http.StatusNotFound, err)
		return
	}

	if err := a.storage.Remove(pathname); err != nil {
		a.renderer.Error(w, http.StatusInternalServerError, err)
		return
	}

	if thumbnailPath, ok := a.thumbnailApp.HasThumbnail(pathname); ok {
		if err := a.storage.Remove(thumbnailPath); err != nil {
			a.renderer.Error(w, http.StatusInternalServerError, err)
			return
		}
	}

	a.List(w, request, r.URL.Query().Get("d"), &provider.Message{Level: "success", Content: fmt.Sprintf("%s successfully deleted", info.Name)})
}
