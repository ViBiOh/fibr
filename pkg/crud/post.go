package crud

import (
	"net/http"

	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/httputils/pkg/errors"
)

// Post handle post from form
func (a *app) Post(w http.ResponseWriter, r *http.Request, request *provider.Request) {
	if r.FormValue("type") == "share" {
		if r.FormValue("method") == http.MethodPost {
			a.CreateShare(w, r, request)
		} else if r.FormValue("method") == http.MethodDelete {
			a.DeleteShare(w, r, request)
		} else {
			a.renderer.Error(w, provider.NewError(http.StatusMethodNotAllowed, errors.New("unknown method")))
		}
	} else if r.FormValue("method") == http.MethodPost {
		a.Upload(w, r, request)
	} else if r.FormValue("method") == http.MethodPatch {
		a.Rename(w, r, request)
	} else if r.FormValue("method") == http.MethodPut {
		a.Create(w, r, request)
	} else if r.FormValue("method") == http.MethodDelete {
		a.Delete(w, r, request)
	} else {
		a.renderer.Error(w, provider.NewError(http.StatusMethodNotAllowed, errors.New("unknown method")))
	}
}
