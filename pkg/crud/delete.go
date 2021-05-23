package crud

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/httputils/v4/pkg/model"
	"github.com/ViBiOh/httputils/v4/pkg/renderer"
)

// Delete given path from filesystem
func (a *app) Delete(w http.ResponseWriter, r *http.Request, request provider.Request) {
	if !request.CanEdit {
		a.rendererApp.Error(w, model.WrapForbidden(ErrNotAuthorized))
		return
	}

	name, err := checkFormName(r, "name")
	if err != nil && !errors.Is(err, ErrEmptyName) {
		a.rendererApp.Error(w, err)
		return
	}

	info, err := a.storageApp.Info(request.GetFilepath(name))
	if err != nil {
		a.rendererApp.Error(w, model.WrapNotFound(err))
		return
	}

	if err := a.storageApp.Remove(info.Pathname); err != nil {
		a.rendererApp.Error(w, model.WrapInternal(err))
		return
	}

	if err := a.metadataApp.DeleteSharePath(info.Pathname); err != nil {
		a.rendererApp.Error(w, model.WrapInternal(err))
		return
	}

	go a.thumbnailApp.Remove(info)

	a.rendererApp.Redirect(w, r, fmt.Sprintf("%s/", request.URL("")), renderer.NewSuccessMessage("%s successfully deleted", info.Name))
}
