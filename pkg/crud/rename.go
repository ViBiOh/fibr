package crud

import (
	"fmt"
	"net/http"

	"github.com/ViBiOh/fibr/pkg/provider"
)

// Rename rename given path to a new one
func (a *app) Rename(w http.ResponseWriter, r *http.Request, request *provider.Request) {
	if !request.CanEdit {
		a.renderer.Error(w, provider.NewError(http.StatusForbidden, ErrNotAuthorized))
		return
	}

	newName, httpErr := checkFormName(r, "newName")
	if httpErr != nil {
		a.renderer.Error(w, httpErr)
		return
	}

	oldPath, httErr := checkFormName(r, "name")
	if httErr != nil && httErr.Err != ErrEmptyName {
		a.renderer.Error(w, httErr)
		return
	}

	newPath := request.GetFilepath(newName)
	oldPath = request.GetFilepath(oldPath)

	if _, err := a.storage.Info(newPath); err == nil {
		a.renderer.Error(w, provider.NewError(http.StatusBadRequest, err))
		return
	} else if !provider.IsNotExist(err) {
		a.renderer.Error(w, provider.NewError(http.StatusInternalServerError, err))
		return
	}

	oldInfo, err := a.storage.Info(oldPath)
	if err != nil {
		if !provider.IsNotExist(err) {
			a.renderer.Error(w, provider.NewError(http.StatusInternalServerError, err))
		} else {
			a.renderer.Error(w, provider.NewError(http.StatusNotFound, err))
		}

		return
	}

	if err := a.storage.Rename(oldPath, newPath); err != nil {
		a.renderer.Error(w, provider.NewError(http.StatusInternalServerError, err))
		return
	}

	newInfo, err := a.storage.Info(newPath)
	if err != nil {
		a.renderer.Error(w, provider.NewError(http.StatusInternalServerError, err))
	}

	go a.renameThumbnail(oldInfo, newInfo)

	a.List(w, request, &provider.Message{Level: "success", Content: fmt.Sprintf("%s successfully renamed to %s", oldInfo.Name, newName)})
}
