package crud

import (
	"fmt"
	"net/http"

	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/fibr/pkg/thumbnail"
)

// Rename rename given path to a new one
func (a *app) Rename(w http.ResponseWriter, r *http.Request, request *provider.Request) {
	if !request.CanEdit {
		a.renderer.Error(w, provider.NewError(http.StatusForbidden, ErrNotAuthorized))
		return
	}

	newName, err := checkFormName(r, "newName")
	if err != nil {
		if err == ErrNotAuthorized {
			a.renderer.Error(w, provider.NewError(http.StatusForbidden, err))
			return
		} else if err == ErrEmptyName {
			a.renderer.Error(w, provider.NewError(http.StatusBadRequest, err))
			return
		}
	}

	newName = request.GetFilepath(newName)
	_, err = a.storage.Info(newName)
	if err == nil {
		a.renderer.Error(w, provider.NewError(http.StatusBadRequest, err))
		return
	} else if !provider.IsNotExist(err) {
		a.renderer.Error(w, provider.NewError(http.StatusInternalServerError, err))
		return
	}

	oldName, err := checkFormName(r, "name")
	if err != nil && err != ErrEmptyName {
		a.renderer.Error(w, provider.NewError(http.StatusForbidden, err))
		return
	}

	oldInfo, err := a.storage.Info(request.GetFilepath(oldName))
	if err != nil {
		if !provider.IsNotExist(err) {
			a.renderer.Error(w, provider.NewError(http.StatusInternalServerError, err))
		} else {
			a.renderer.Error(w, provider.NewError(http.StatusNotFound, err))
		}

		return
	}

	if err := a.storage.Rename(oldInfo.Pathname, newName); err != nil {
		a.renderer.Error(w, provider.NewError(http.StatusInternalServerError, err))
		return
	}

	if thumbnailPath, ok := a.thumbnail.HasThumbnail(oldName); ok {
		if err := a.storage.Remove(thumbnailPath); err != nil {
			a.renderer.Error(w, provider.NewError(http.StatusInternalServerError, err))
			return
		}

		if thumbnail.CanHaveThumbnail(newName) {
			a.thumbnail.AsyncGenerateThumbnail(newName)
		}
	}

	a.List(w, request, r.URL.Query().Get("d"), &provider.Message{Level: "success", Content: fmt.Sprintf("%s successfully renamed to %s", oldInfo.Name, newName)})
}
