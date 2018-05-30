package crud

import (
	"fmt"
	"net/http"
	"path"

	"github.com/ViBiOh/fibr/pkg/provider"
)

// Delete given path from filesystem
func (a *App) Delete(w http.ResponseWriter, r *http.Request, config *provider.Request) {
	if !config.CanEdit {
		a.renderer.Error(w, http.StatusForbidden, ErrNotAuthorized)
		return
	}

	pathname, err := getFilepath(r, config)
	if err != nil && err == ErrNotAuthorized {
		a.renderer.Error(w, http.StatusForbidden, err)
		return
	}

	info, err := a.storage.Info(pathname)
	if err != nil {
		a.renderer.Error(w, http.StatusNotFound, fmt.Errorf(`Requested path does not exist %s`, pathname))
		return
	}

	if err := a.storage.Remove(pathname); err != nil {
		a.renderer.Error(w, http.StatusInternalServerError, fmt.Errorf(`Error while deleting %s: %v`, pathname, err))
		return
	}

	a.List(w, config, path.Dir(pathname), r.URL.Query().Get(`d`), &provider.Message{Level: `success`, Content: fmt.Sprintf(`%s successfully deleted`, info.Name)})
}
