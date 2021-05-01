package crud

import (
	"fmt"
	"net/http"

	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/httputils/v4/pkg/renderer"
)

// Delete given path from filesystem
func (a *app) Delete(w http.ResponseWriter, r *http.Request, request provider.Request) {
	if !request.CanEdit {
		a.rendererApp.Error(w, request, provider.NewError(http.StatusForbidden, ErrNotAuthorized))
		return
	}

	name, httpErr := checkFormName(r, "name")
	if httpErr != nil && httpErr.Err != ErrEmptyName {
		a.rendererApp.Error(w, request, httpErr)
		return
	}

	info, err := a.storageApp.Info(request.GetFilepath(name))
	if err != nil {
		a.rendererApp.Error(w, request, provider.NewError(http.StatusNotFound, err))
		return
	}

	if err := a.storageApp.Remove(info.Pathname); err != nil {
		a.rendererApp.Error(w, request, provider.NewError(http.StatusInternalServerError, err))
		return
	}

	if err := a.metadataApp.DeleteSharePath(info.Pathname); err != nil {
		a.rendererApp.Error(w, request, provider.NewError(http.StatusInternalServerError, err))
		return
	}

	go a.thumbnailApp.Remove(info)

	http.Redirect(w, r, fmt.Sprintf("%s/?%s", request.GetURI(""), renderer.NewSuccessMessage(fmt.Sprintf("%s successfully deleted", info.Name))), http.StatusFound)
}
