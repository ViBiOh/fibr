package crud

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"path"

	"github.com/ViBiOh/fibr/provider"
)

// Delete given path from filesystem
func (a *App) Delete(w http.ResponseWriter, r *http.Request, config *provider.RequestConfig) {
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

	if err := os.RemoveAll(filename); err != nil {
		a.renderer.Error(w, http.StatusInternalServerError, fmt.Errorf(`Error while deleting %s: %v`, filename, err))
		return
	}

	a.GetDir(w, config, path.Dir(filename), r.URL.Query().Get(`d`), &provider.Message{Level: `success`, Content: fmt.Sprintf(`%s successfully deleted`, info.Name())})
}
