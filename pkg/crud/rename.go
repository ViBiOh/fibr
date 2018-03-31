package crud

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"path"

	"github.com/ViBiOh/fibr/pkg/provider"
)

// Rename rename given path to a new one
func (a *App) Rename(w http.ResponseWriter, r *http.Request, config *provider.RequestConfig) {
	if !config.CanEdit {
		a.renderer.Error(w, http.StatusForbidden, ErrNotAuthorized)
		return
	}

	filename, info, err := a.getFormOrPathFilename(r, config)
	if err != nil && err == ErrNotAuthorized {
		a.renderer.Error(w, http.StatusForbidden, err)
		return
	} else if info == nil {
		a.renderer.Error(w, http.StatusNotFound, errors.New(`Requested path does not exist`))
		return
	}

	newFilename := r.URL.Query().Get(`newName`)

	if err := os.Rename(filename, newFilename); err != nil {
		a.renderer.Error(w, http.StatusInternalServerError, fmt.Errorf(`Error while renaming file: %v`, err))
		return
	}

	a.GetDir(w, config, path.Dir(newFilename), r.URL.Query().Get(`d`), &provider.Message{Level: `success`, Content: fmt.Sprintf(`%s successfully renamed to %s`, info.Name(), newFilename)})
}
