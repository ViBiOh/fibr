package crud

import (
	"fmt"
	"net/http"

	"github.com/ViBiOh/fibr/pkg/provider"
)

// Rename rename given path to a new one
func (a *App) Rename(w http.ResponseWriter, r *http.Request, request *provider.Request) {
	if !request.CanEdit {
		a.renderer.Error(w, http.StatusForbidden, ErrNotAuthorized)
		return
	}

	newName, err := getFormFilepath(r, request, `newName`)
	if err != nil {
		if err == ErrNotAuthorized {
			a.renderer.Error(w, http.StatusForbidden, err)
			return
		} else if err == ErrEmptyName {
			a.renderer.Error(w, http.StatusBadRequest, err)
			return
		}
	}

	_, err = a.storage.Info(newName)
	if err == nil {
		a.renderer.Error(w, http.StatusBadRequest, err)
		return
	} else if !provider.IsNotExist(err) {
		a.renderer.Error(w, http.StatusInternalServerError, err)
		return
	}

	oldName, err := getFilepath(r, request)
	if err != nil && err == ErrNotAuthorized {
		a.renderer.Error(w, http.StatusForbidden, err)
		return
	}

	info, err := a.storage.Info(oldName)
	if err != nil {
		if !provider.IsNotExist(err) {
			a.renderer.Error(w, http.StatusInternalServerError, err)
		} else {
			a.renderer.Error(w, http.StatusNotFound, err)
		}

		return
	}

	if err := a.storage.Rename(oldName, newName); err != nil {
		a.renderer.Error(w, http.StatusInternalServerError, err)
		return
	}

	a.List(w, request, r.URL.Query().Get(`d`), &provider.Message{Level: `success`, Content: fmt.Sprintf(`%s successfully renamed to %s`, info.Name, newName)})
}
