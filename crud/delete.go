package crud

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"path"

	"github.com/ViBiOh/fibr/provider"
	"github.com/ViBiOh/fibr/utils"
)

// Delete given path from filesystem
func (a *App) Delete(w http.ResponseWriter, r *http.Request, config *provider.RequestConfig) {
	if !config.CanEdit || path.Join(config.Root, config.Path) == `/` {
		a.renderer.Error(w, http.StatusForbidden, errors.New(`You're not authorized to do this ⛔`))
	} else if filename, info := utils.GetPathInfo(config.Root, config.Path); info == nil {
		a.renderer.Error(w, http.StatusNotFound, errors.New(`Requested path does not exist`))
	} else if err := os.RemoveAll(filename); err != nil {
		a.renderer.Error(w, http.StatusInternalServerError, fmt.Errorf(`Error while deleting: %v`, err))
	} else {
		w.WriteHeader(http.StatusNoContent)
	}
}
