package crud

import (
	"fmt"
	"net/http"
	"os"
	"path"
	"strings"

	"github.com/ViBiOh/fibr/pkg/provider"
)

// Create creates given path directory to filesystem
func (a *App) Create(w http.ResponseWriter, r *http.Request, request *provider.Request) {
	if !request.CanEdit {
		a.renderer.Error(w, http.StatusForbidden, ErrNotAuthorized)
		return
	}

	var filename string

	formName := r.FormValue(`name`)
	if formName != `` {
		filename, _ = a.getFileinfo(request, []byte(formName))
	}

	if filename == `` {
		if !strings.HasSuffix(request.Path, `/`) {
			a.renderer.Error(w, http.StatusForbidden, ErrNotAuthorized)
			return
		}

		filename, _ = a.getFileinfo(request, nil)
	}

	if strings.Contains(filename, `..`) {
		a.renderer.Error(w, http.StatusForbidden, ErrNotAuthorized)
		return
	}

	if err := os.MkdirAll(filename, 0700); err != nil {
		a.renderer.Error(w, http.StatusInternalServerError, fmt.Errorf(`Error while creating directory: %v`, err))
		return
	}

	a.GetDir(w, request, path.Dir(filename), r.URL.Query().Get(`d`), &provider.Message{Level: `success`, Content: fmt.Sprintf(`Directory %s successfully created`, path.Base(filename))})
}
