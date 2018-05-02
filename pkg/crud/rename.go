package crud

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"path"
	"strings"

	"github.com/ViBiOh/fibr/pkg/provider"
)

// Rename rename given path to a new one
func (a *App) Rename(w http.ResponseWriter, r *http.Request, request *provider.Request) {
	if !request.CanEdit {
		a.renderer.Error(w, http.StatusForbidden, ErrNotAuthorized)
		return
	}

	newName := r.FormValue(`newName`)
	if strings.TrimSpace(newName) == `` {
		a.renderer.Error(w, http.StatusBadRequest, errors.New(`New name is empty`))
	}

	filename, info, err := a.getFormOrPathFilename(r, request)
	if err != nil && err == ErrNotAuthorized {
		a.renderer.Error(w, http.StatusForbidden, err)
		return
	} else if info == nil {
		a.renderer.Error(w, http.StatusNotFound, errors.New(`Requested path does not exist`))
		return
	}

	newFilename, newInfo := a.getFileinfo(request, []byte(newName))
	if newInfo != nil {
		a.renderer.Error(w, http.StatusBadRequest, fmt.Errorf(`%s already exists`, newName))
		return
	}

	if err := os.Rename(filename, newFilename); err != nil {
		a.renderer.Error(w, http.StatusInternalServerError, fmt.Errorf(`Error while renaming file: %v`, err))
		return
	}

	a.GetDir(w, request, path.Dir(newFilename), r.URL.Query().Get(`d`), &provider.Message{Level: `success`, Content: fmt.Sprintf(`%s successfully renamed to %s`, info.Name(), newName)})
}
