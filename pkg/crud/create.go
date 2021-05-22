package crud

import (
	"net/http"
	"path"

	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/httputils/v4/pkg/renderer"
)

// Create creates given path directory to filesystem
func (a *app) Create(w http.ResponseWriter, r *http.Request, request provider.Request) {
	if !request.CanEdit {
		a.rendererApp.Error(w, provider.NewError(http.StatusForbidden, ErrNotAuthorized))
		return
	}

	name, httpErr := checkFormName(r, "name")
	if httpErr != nil && httpErr.Err != ErrEmptyName {
		a.rendererApp.Error(w, httpErr)
		return
	}

	name, err := provider.SanitizeName(name, false)
	if err != nil {
		a.rendererApp.Error(w, provider.NewError(http.StatusInternalServerError, err))
		return
	}

	pathname := request.GetFilepath(name)

	if err := a.storageApp.CreateDir(pathname); err != nil {
		a.rendererApp.Error(w, provider.NewError(http.StatusInternalServerError, err))
		return
	}

	a.rendererApp.Redirect(w, r, request.GetURI(name), renderer.NewSuccessMessage("Directory %s successfully created", path.Base(pathname)))
}
