package renderer

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/httputils/v3/pkg/httperror"
	"github.com/ViBiOh/httputils/v3/pkg/logger"
	"github.com/ViBiOh/httputils/v3/pkg/templates"
)

func (a app) newPageBuilder() *provider.PageBuilder {
	return (&provider.PageBuilder{}).Config(a.config)
}

func findIndex(arr []string, value string) int {
	for index, item := range arr {
		if item == value {
			return index
		}
	}

	return -1
}

func removeIndex(arr []string, index int) []string {
	if len(arr) == 0 || index < 0 || index >= len(arr) {
		return arr
	}

	return append(arr[:index], arr[index+1:]...)
}

func computeListLayoutPaths(request provider.Request, page provider.Page) string {
	listLayoutPaths := request.Preferences.ListLayoutPath

	switch page.Layout {
	case "list":
		if index := findIndex(listLayoutPaths, request.Path); index == -1 {
			listLayoutPaths = append(listLayoutPaths, request.Path)
		}
	case "grid":
		listLayoutPaths = removeIndex(listLayoutPaths, findIndex(listLayoutPaths, request.Path))
	}

	return strings.Join(listLayoutPaths, ",")
}

func setPrefsCookie(w http.ResponseWriter, request provider.Request, page provider.Page) {
	http.SetCookie(w, &http.Cookie{
		Name:     "list_layout_paths",
		Value:    computeListLayoutPaths(request, page),
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	})
}

// Directory render directory listing
func (a app) Directory(w http.ResponseWriter, request provider.Request, content map[string]interface{}, message *provider.Message) {
	page := a.newPageBuilder().Request(request).Message(message).Layout(request.Display).Content(content).Build()

	w.Header().Set("content-language", "en")
	setPrefsCookie(w, request, page)

	if err := templates.ResponseHTMLTemplate(a.tpl.Lookup("files"), w, page, http.StatusOK); err != nil {
		a.Error(w, request, provider.NewError(http.StatusInternalServerError, err))
		return
	}
}

// File render file detail
func (a app) File(w http.ResponseWriter, request provider.Request, content map[string]interface{}, message *provider.Message) {
	page := a.newPageBuilder().Request(request).Message(message).Layout("browser").Content(content).Build()

	w.Header().Set("content-language", "en")
	setPrefsCookie(w, request, page)

	if err := templates.ResponseHTMLTemplate(a.tpl.Lookup("file"), w, page, http.StatusOK); err != nil {
		a.Error(w, request, provider.NewError(http.StatusInternalServerError, err))
		return
	}
}

// Error render error page with given status
func (a app) Error(w http.ResponseWriter, request provider.Request, err *provider.Error) {
	logger.Error("%s", err.Err)

	if err.Status == http.StatusUnauthorized {
		w.Header().Add("WWW-Authenticate", `Basic realm="Password required" charset="UTF-8"`)
	}

	page := a.newPageBuilder().Request(request).Error(err).Build()

	if err := templates.ResponseHTMLTemplate(a.tpl.Lookup("error"), w, page, err.Status); err != nil {
		httperror.InternalServerError(w, err)
		return
	}
}

// Sitemap render sitemap.xml
func (a app) Sitemap(w http.ResponseWriter) {
	if err := templates.ResponseXMLTemplate(a.tpl.Lookup("sitemap"), w, a.newPageBuilder().Build(), http.StatusOK); err != nil {
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

	if err := templates.WriteTemplate(tpl, w, fill, "text/xml"); err != nil {
		httperror.InternalServerError(w, err)
		return
	}
}
