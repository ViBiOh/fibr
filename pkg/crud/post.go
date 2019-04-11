package crud

import (
	"net/http"

	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/httputils/pkg/errors"
)

// Post handle post from form
func (a *App) Post(w http.ResponseWriter, r *http.Request, config *provider.Request) {
	if r.FormValue("type") == "share" {
		if r.FormValue("method") == http.MethodPost {
			a.CreateShare(w, r, config)
		} else if r.FormValue("method") == http.MethodDelete {
			a.DeleteShare(w, r, config)
		} else {
			a.renderer.Error(w, http.StatusMethodNotAllowed, errors.New("unknown method"))
		}
	} else if r.FormValue("method") == http.MethodPost {
		a.Upload(w, r, config)
	} else if r.FormValue("method") == http.MethodPatch {
		a.Rename(w, r, config)
	} else if r.FormValue("method") == http.MethodPut {
		a.Create(w, r, config)
	} else if r.FormValue("method") == http.MethodDelete {
		a.Delete(w, r, config)
	} else {
		a.renderer.Error(w, http.StatusMethodNotAllowed, errors.New("unknown method"))
	}
}
