package renderer

import (
	"fmt"
	"net/http"

	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/httputils/pkg/errors"
	"github.com/ViBiOh/httputils/pkg/httperror"
	"github.com/ViBiOh/httputils/pkg/logger"
	"github.com/ViBiOh/httputils/pkg/templates"
)

// Error render error page with given status
func (a app) Error(w http.ResponseWriter, err *provider.Error) {
	logger.Error("%#v", err.Err)

	if err.Status == http.StatusUnauthorized {
		w.Header().Add("WWW-Authenticate", "Basic realm=\"Password required\" charset=\"UTF-8\"")
	}

	page := a.newPageBuilder().Error(err).Build()

	if err := templates.WriteHTMLTemplate(a.tpl.Lookup("error"), w, page, err.Status); err != nil {
		httperror.InternalServerError(w, err)
		return
	}
}

// Sitemap render sitemap.xml
func (a app) Sitemap(w http.ResponseWriter) {
	page := a.newPageBuilder().Build()

	if err := templates.WriteXMLTemplate(a.tpl.Lookup("sitemap"), w, page, http.StatusOK); err != nil {
		httperror.InternalServerError(w, err)
		return
	}
}

// SVG render a svg in given coolor
func (a app) SVG(w http.ResponseWriter, name, fill string) {
	tpl := a.tpl.Lookup(fmt.Sprintf("svg-%s", name))
	if tpl == nil {
		httperror.NotFound(w)
		return
	}

	w.Header().Set("Content-Type", "image/svg+xml")

	if err := tpl.Execute(w, fill); err != nil {
		httperror.InternalServerError(w, errors.WithStack(err))
		return
	}
}
