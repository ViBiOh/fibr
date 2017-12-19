package crud

import (
	"fmt"
	"net/http"
	"os"

	"github.com/ViBiOh/fibr/utils"
	"github.com/ViBiOh/httputils"
)

// Delete given path from filesystem
func Delete(w http.ResponseWriter, r *http.Request, directory string) {
	if r.URL.Path == `/` {
		httputils.Forbidden(w)
	} else if filename, info := utils.GetPathInfo(directory, r.URL.Path); info == nil {
		httputils.NotFound(w)
	} else if err := os.RemoveAll(filename); err != nil {
		httputils.InternalServerError(w, fmt.Errorf(`Error while deleting: %v`, err))
	} else {
		w.WriteHeader(http.StatusNoContent)
	}
}
