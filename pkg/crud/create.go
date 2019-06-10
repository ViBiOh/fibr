package crud

import (
	"fmt"
	"net/http"
	"net/url"
	"path"

	"github.com/ViBiOh/fibr/pkg/provider"
)

// Create creates given path directory to filesystem
func (a *app) Create(w http.ResponseWriter, r *http.Request, request *provider.Request) {
	if !request.CanEdit {
		a.renderer.Error(w, provider.NewError(http.StatusForbidden, ErrNotAuthorized))
		return
	}

	name, httpErr := checkFormName(r, "name")
	if httpErr != nil && httpErr.Err != ErrEmptyName {
		a.renderer.Error(w, httpErr)
		return
	}

	pathname, err := provider.SanitizeName(request.GetFilepath(name), false)
	if err != nil {
		a.renderer.Error(w, provider.NewError(http.StatusInternalServerError, err))
		return
	}

	if err := a.storage.CreateDir(pathname); err != nil {
		a.renderer.Error(w, provider.NewError(http.StatusInternalServerError, err))
		return
	}

	http.Redirect(w, r, fmt.Sprintf("%s/?message=%s&messageLevel=success", pathname, url.QueryEscape(fmt.Sprintf("Directory %s successfully created", path.Base(pathname)))), http.StatusMovedPermanently)
}
