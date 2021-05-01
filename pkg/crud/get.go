package crud

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/httputils/v4/pkg/query"
	"github.com/ViBiOh/httputils/v4/pkg/renderer"
)

var (
	staticRootPath = []string{
		"/robots.txt",
		"/browserconfig.xml",
		"/favicon.ico",
	}
)

// ServeStatic check if filename match SEO or static filename and serve it
func (a *app) ServeStatic(w http.ResponseWriter, r *http.Request) bool {
	if r.Method != http.MethodGet {
		return false
	}

	if r.URL.Path == "/sitemap.xml" {
		a.rendererApp.Sitemap(w)
		return true
	}

	if strings.HasPrefix(r.URL.Path, "/svg") {
		a.rendererApp.SVG(w, strings.TrimPrefix(r.URL.Path, "/svg/"), r.URL.Query().Get("fill"))
		return true
	}

	if strings.HasPrefix(r.URL.Path, "/favicon") {
		a.staticHandler.ServeHTTP(w, r)
		return true
	}

	for _, staticPath := range staticRootPath {
		if r.URL.Path == staticPath {
			a.staticHandler.ServeHTTP(w, r)
			return true
		}
	}

	return false
}

func (a *app) getWithMessage(w http.ResponseWriter, r *http.Request, request provider.Request, message renderer.Message) {
	info, err := a.storageApp.Info(request.GetFilepath(""))
	if err != nil {
		if provider.IsNotExist(err) {
			a.rendererApp.Error(w, request, provider.NewError(http.StatusNotFound, err))
		} else {
			a.rendererApp.Error(w, request, provider.NewError(http.StatusInternalServerError, err))
		}
		return
	}

	if query.GetBool(r, "thumbnail") {
		if info.IsDir {
			a.thumbnailApp.List(w, r, info)
		} else {
			a.thumbnailApp.Serve(w, r, info)
		}

		return
	}

	if !info.IsDir {
		if query.GetBool(r, "browser") {
			a.Browser(w, request, info, message)
		} else if file, err := a.storageApp.ReaderFrom(info.Pathname); err != nil {
			a.rendererApp.Error(w, request, provider.NewError(http.StatusInternalServerError, err))
		} else {
			http.ServeContent(w, r, info.Name, info.Date, file)
		}

		return
	}

	if query.GetBool(r, "download") {
		a.Download(w, request)
		return
	}

	if !strings.HasSuffix(r.URL.Path, "/") {
		http.Redirect(w, r, fmt.Sprintf("%s/", r.URL.Path), http.StatusPermanentRedirect)
		return
	}

	a.List(w, request, message)
}

// Get output content
func (a *app) Get(w http.ResponseWriter, r *http.Request, request provider.Request) {
	a.getWithMessage(w, r, request, renderer.ParseMessage(r))
}
