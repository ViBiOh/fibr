package crud

import (
	"errors"
	"net/http"

	"github.com/ViBiOh/fibr/provider"
)

// Post handle post from form
func (a *App) Post(w http.ResponseWriter, r *http.Request, config *provider.RequestConfig) {
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
