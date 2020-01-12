package crud

import (
	"fmt"
	"net/http"

	"github.com/ViBiOh/fibr/pkg/provider"
)

// Post handle post from form
func (a *app) Post(w http.ResponseWriter, r *http.Request, request provider.Request) {
	method := r.FormValue("method")

	if r.FormValue("type") == "share" {
		switch method {
		case http.MethodPost:
			a.CreateShare(w, r, request)
		case http.MethodDelete:
			a.DeleteShare(w, r, request)
		default:
			a.renderer.Error(w, provider.NewError(http.StatusMethodNotAllowed, fmt.Errorf("unknown method %s for %s: %#v", method, r.URL.Path, request)))
		}

		return
	}

	switch method {
	case http.MethodPost:
		a.Upload(w, r, request)
	case http.MethodPatch:
		a.Rename(w, r, request)
	case http.MethodPut:
		a.Create(w, r, request)
	case http.MethodDelete:
		a.Delete(w, r, request)
	default:
		a.renderer.Error(w, provider.NewError(http.StatusMethodNotAllowed, fmt.Errorf("unknown method %s for %s: %#v", method, r.URL.Path, request)))
	}
}
