package crud

import (
	"fmt"
	"net/http"
	"path"
	"strings"

	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/httputils/pkg/query"
)

// ServeStatic check if filename match SEO or static filename and serve it
func (a *app) ServeStatic(w http.ResponseWriter, r *http.Request) bool {
	if r.Method != http.MethodGet {
		return false
	}

	if r.URL.Path == "/robots.txt" || strings.HasPrefix(r.URL.Path, "/favicon") {
		http.ServeFile(w, r, path.Join("templates/static", r.URL.Path))
		return true
	}

	if r.URL.Path == "/sitemap.xml" {
		a.renderer.Sitemap(w)
		return true
	}

	if strings.HasPrefix(r.URL.Path, "/svg") {
		a.renderer.SVG(w, strings.TrimPrefix(r.URL.Path, "/svg/"), r.URL.Query().Get("fill"))
		return true
	}

	return false
}

func isThumbnail(r *http.Request) bool {
	return query.GetBool(r, "thumbnail")
}

// GetWithMessage output content with given message
func (a *app) GetWithMessage(w http.ResponseWriter, r *http.Request, request *provider.Request, message *provider.Message) {
	info, err := a.storage.Info(request.GetFilepath(""))
	if err != nil {
		if provider.IsNotExist(err) {
			a.renderer.Error(w, provider.NewError(http.StatusNotFound, err))
		} else {
			a.renderer.Error(w, provider.NewError(http.StatusInternalServerError, err))
		}
		return
	}

	if isThumbnail(r) {
		if info.IsDir {
			a.thumbnail.List(w, r, info.Pathname)
			return
		}

		if a.thumbnail.Serve(w, r, info.Pathname) {
			return
		}
	}

	if !info.IsDir {
		if query.GetBool(r, "browser") {
			a.Browser(w, request, info, message)
		} else {
			a.storage.Serve(w, r, info.Pathname)
		}

		return
	}

	if !strings.HasSuffix(r.URL.Path, "/") {
		http.Redirect(w, r, fmt.Sprintf("%s/", r.URL.Path), http.StatusPermanentRedirect)
		return
	}

	a.List(w, request, r.URL.Query().Get("d"), message)
}

// Get output content
func (a *app) Get(w http.ResponseWriter, r *http.Request, request *provider.Request) {
	var message *provider.Message

	if messageContent := strings.TrimSpace(r.URL.Query().Get("message")); messageContent != "" {
		message = &provider.Message{
			Level:   r.URL.Query().Get("messageLevel"),
			Content: messageContent,
		}
	}

	a.GetWithMessage(w, r, request, message)
}
