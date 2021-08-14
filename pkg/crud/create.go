package crud

import (
	"errors"
	"fmt"
	"net/http"
	"path"

	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/httputils/v4/pkg/model"
	"github.com/ViBiOh/httputils/v4/pkg/renderer"
)

// Create creates given path directory to filesystem
func (a App) Create(w http.ResponseWriter, r *http.Request, request provider.Request) {
	if !request.CanEdit {
		a.rendererApp.Error(w, model.WrapForbidden(ErrNotAuthorized))
		return
	}

	name, err := checkFormName(r, "name")
	if err != nil && !errors.Is(err, ErrEmptyName) {
		a.rendererApp.Error(w, err)
		return
	}

	name, err = provider.SanitizeName(name, false)
	if err != nil {
		a.rendererApp.Error(w, model.WrapInternal(err))
		return
	}

	pathname := request.GetFilepath(name)

	if err := a.storageApp.CreateDir(pathname); err != nil {
		a.rendererApp.Error(w, model.WrapInternal(err))
		return
	}

	a.rendererApp.Redirect(w, r, fmt.Sprintf("%s/?d=%s", request.URL(name), request.Layout("")), renderer.NewSuccessMessage("Directory %s successfully created", path.Base(pathname)))
}
