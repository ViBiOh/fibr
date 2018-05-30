package crud

import (
	"fmt"
	"net/http"
	"path"

	"github.com/ViBiOh/fibr/pkg/provider"
)

// Rename rename given path to a new one
func (a *App) Rename(w http.ResponseWriter, r *http.Request, config *provider.Request) {
	if !config.CanEdit {
		a.renderer.Error(w, http.StatusForbidden, ErrNotAuthorized)
		return
	}

	newName, err := getFormFilepath(r, config, `newName`)
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
		a.renderer.Error(w, http.StatusBadRequest, fmt.Errorf(`%s already exists`, newName))
		return
	} else if !provider.IsNotExist(err) {
		a.renderer.Error(w, http.StatusInternalServerError, fmt.Errorf(`Error while getting infos for %s: %v`, newName, err))
		return
	}

	oldName, err := getFilepath(r, config)
	if err != nil && err == ErrNotAuthorized {
		a.renderer.Error(w, http.StatusForbidden, err)
		return
	}

	info, err := a.storage.Info(oldName)
	if err != nil {
		if !provider.IsNotExist(err) {
			a.renderer.Error(w, http.StatusInternalServerError, fmt.Errorf(`Error while getting infos for %s: %v`, oldName, err))
		} else {
			a.renderer.Error(w, http.StatusNotFound, fmt.Errorf(`Requested path does not exist %s`, oldName))
		}

		return
	}

	if err := a.storage.Rename(oldName, newName); err != nil {
		a.renderer.Error(w, http.StatusInternalServerError, fmt.Errorf(`Error while renaming file: %v`, err))
		return
	}

	a.List(w, config, path.Dir(oldName), r.URL.Query().Get(`d`), &provider.Message{Level: `success`, Content: fmt.Sprintf(`%s successfully renamed to %s`, info.Name, newName)})
}
