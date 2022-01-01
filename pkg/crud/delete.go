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
func (a App) Delete(w http.ResponseWriter, r *http.Request, request provider.Request) {
	if !request.CanEdit {
		a.error(w, r, request, model.WrapForbidden(ErrNotAuthorized))
		return
	}

	name, err := checkFormName(r, "name")
	if err != nil && !errors.Is(err, ErrEmptyName) {
		a.error(w, r, request, err)
		return
	}

	pathname := request.SubPath(name)
	info, err := a.storageApp.Info(pathname)
	if err != nil {
		a.error(w, r, request, model.WrapNotFound(err))
		return
	}

	if err = a.storageApp.Remove(info.Pathname); err != nil {
		a.error(w, r, request, model.WrapInternal(err))
		return
	}

	if info.IsDir {
		provider.SetPrefsCookie(w, deletePreferences(request, pathname))
	}

	go a.notify(provider.NewDeleteEvent(request, info, a.rendererApp))

	a.rendererApp.Redirect(w, r, fmt.Sprintf("?d=%s", request.Display), renderer.NewSuccessMessage("%s successfully deleted", info.Name))
}

func deletePreferences(request provider.Request, oldPath string) provider.Request {
	var paths []string

	for _, layoutPath := range request.Preferences.ListLayoutPath {
		if layoutPath != oldPath {
			paths = append(paths, layoutPath)
		}
	}

	request.Preferences.ListLayoutPath = paths
	return request
}
