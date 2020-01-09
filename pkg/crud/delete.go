package crud

import (
	"fmt"
	"net/http"

	"github.com/ViBiOh/fibr/pkg/provider"
)

// Delete given path from filesystem
func (a *app) Delete(w http.ResponseWriter, r *http.Request, request provider.Request) {
	if !request.CanEdit {
		a.renderer.Error(w, provider.NewError(http.StatusForbidden, ErrNotAuthorized))
		return
	}

	name, httpErr := checkFormName(r, "name")
	if httpErr != nil && httpErr.Err != ErrEmptyName {
		a.renderer.Error(w, httpErr)
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

	go a.deleteThumbnail(info)

	a.List(w, request, &provider.Message{Level: "success", Content: fmt.Sprintf("%s successfully deleted", info.Name)})
}
