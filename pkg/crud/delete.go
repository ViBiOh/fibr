package crud

import (
	"fmt"
	"net/http"

	"github.com/ViBiOh/fibr/pkg/provider"
)

// Delete given path from filesystem
func (a *app) Delete(w http.ResponseWriter, r *http.Request, request *provider.Request) {
	if !request.CanEdit {
		a.renderer.Error(w, provider.NewError(http.StatusForbidden, ErrNotAuthorized))
		return
	}

	name, err := checkFormName(r, "name")
	if err != nil && err != ErrEmptyName {
		a.renderer.Error(w, provider.NewError(http.StatusForbidden, err))
		return
	}

	info, err := a.storage.Info(request.GetFilepath(name))
	if err != nil {
		a.renderer.Error(w, provider.NewError(http.StatusNotFound, err))
		return
	}

	if err := a.storage.Remove(info.Pathname); err != nil {
		a.renderer.Error(w, provider.NewError(http.StatusInternalServerError, err))
		return
	}

	if thumbnailPath, ok := a.thumbnail.HasThumbnail(info.Pathname); ok {
		if err := a.storage.Remove(thumbnailPath); err != nil {
			a.renderer.Error(w, provider.NewError(http.StatusInternalServerError, err))
			return
		}
	}

	a.List(w, request, r.URL.Query().Get("d"), &provider.Message{Level: "success", Content: fmt.Sprintf("%s successfully deleted", info.Name)})
}
