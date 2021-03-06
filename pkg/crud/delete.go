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

	oldPath := request.GetFilepath(name)
	info, err := a.storageApp.Info(oldPath)
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

	if info.IsDir {
		provider.SetPrefsCookie(w, deletePreferences(request, oldPath))
	}

	go a.thumbnailApp.Remove(info)

	a.rendererApp.Redirect(w, r, fmt.Sprintf("%s/?d=%s", request.URL(""), request.Layout("")), renderer.NewSuccessMessage("%s successfully deleted", info.Name))
}

func deletePreferences(request provider.Request, oldPath string) provider.Request {
	paths := make([]string, 0)

	for _, layoutPath := range request.Preferences.ListLayoutPath {
		if layoutPath != oldPath {
			paths = append(paths, layoutPath)
		}
	}

	request.Preferences.ListLayoutPath = paths
	return request
}
