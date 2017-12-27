package crud

import (
	"errors"
	"fmt"
	"net/http"
	"os"

	"github.com/ViBiOh/fibr/ui"
	"github.com/ViBiOh/fibr/utils"
)

// Delete given path from filesystem
func Delete(w http.ResponseWriter, r *http.Request, directory string, uiConfig *ui.Config) {
	if r.URL.Path == `/` {
		uiConfig.Error(w, http.StatusForbidden, errors.New(`You're not authorized to do this ⛔`))
	} else if filename, info := utils.GetPathInfo(directory, r.URL.Path); info == nil {
		uiConfig.Error(w, http.StatusNotFound, errors.New(`Requested path does not exist`))
	} else if err := os.RemoveAll(filename); err != nil {
		uiConfig.Error(w, http.StatusInternalServerError, fmt.Errorf(`Error while deleting: %v`, err))
	} else {
		w.WriteHeader(http.StatusNoContent)
	}
}
