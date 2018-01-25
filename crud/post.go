package crud

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/ViBiOh/fibr/provider"
	"github.com/ViBiOh/httputils"
)

const maxPostMemory = 32 * 1024 * 2014 // 32 MB

// Post handle post from form
func (a *App) Post(w http.ResponseWriter, r *http.Request, config *provider.RequestConfig) {
	if err := r.ParseMultipartForm(maxPostMemory); err != nil {
		httputils.BadRequest(w, fmt.Errorf(`Error while parsing form: %v`, err))
	}

	if r.FormValue(`type`) == `share` {
		if r.FormValue(`method`) == http.MethodPost {
			a.CreateShare(w, r, config)
		} else if r.FormValue(`method`) == http.MethodDelete {
			a.DeleteShare(w, r, config)
		} else {
			a.renderer.Error(w, http.StatusMethodNotAllowed, errors.New(`Unknown method`))
		}
	} else if r.FormValue(`method`) == http.MethodPost {
		a.SaveFile(w, r, config)
	} else if r.FormValue(`method`) == http.MethodPut {
		a.CreateDir(w, r, config)
	} else if r.FormValue(`method`) == http.MethodDelete {
		a.Delete(w, r, config)
	} else {
		a.renderer.Error(w, http.StatusMethodNotAllowed, errors.New(`Unknown method`))
	}
}
