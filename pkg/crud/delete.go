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
		a.rendererApp.Error(w, r, model.WrapForbidden(ErrNotAuthorized))
		return
	}

	name, err := checkFormName(r, "name")
	if err != nil && !errors.Is(err, ErrEmptyName) {
		a.rendererApp.Error(w, r, err)
		return
	}

	pathname := request.GetFilepath(name)
	info, err := a.storageApp.Info(pathname)
	if err != nil {
		a.rendererApp.Error(w, r, model.WrapNotFound(err))
		return
	}

	if err = a.storageApp.Remove(info.Pathname); err != nil {
		a.rendererApp.Error(w, r, model.WrapInternal(err))
		return
	}

	if info.IsDir {
		provider.SetPrefsCookie(w, deletePreferences(request, pathname))
	}

	go a.notify(provider.NewDeleteEvent(info))

	a.rendererApp.Redirect(w, r, fmt.Sprintf("%s/?d=%s", request.URL(""), request.Layout("")), renderer.NewSuccessMessage("%s successfully deleted", info.Name))
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
