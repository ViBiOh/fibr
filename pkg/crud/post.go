package crud

import (
	"errors"
	"net/http"

	"github.com/ViBiOh/fibr/pkg/provider"
)

// Post handle post from form
func (a *app) Post(w http.ResponseWriter, r *http.Request, request *provider.Request) {
	if r.FormValue("type") == "share" {
		switch r.FormValue("method") {
		case http.MethodPost:
			a.CreateShare(w, r, request)
		case http.MethodDelete:
			a.DeleteShare(w, r, request)
		default:
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
