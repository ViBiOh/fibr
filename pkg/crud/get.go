package crud

import (
	"fmt"
	"net/http"
	"path"
	"strings"

	"github.com/ViBiOh/fibr/pkg/provider"
)

// CheckAndServeSEO check if filename match SEO and serve it, or not
func (a *App) CheckAndServeSEO(w http.ResponseWriter, r *http.Request) bool {
	if r.Method != http.MethodGet {
		return false
	}

	if r.URL.Path == `/robots.txt` || strings.HasPrefix(r.URL.Path, `/favicon`) {
		http.ServeFile(w, r, path.Join(`templates/static`, r.URL.Path))
		return true
	}

	if r.URL.Path == `/sitemap.xml` {
		a.renderer.Sitemap(w)
		return true
	}

	return false
}

func (a *App) checkAndServeThumbnail(w http.ResponseWriter, r *http.Request, pathname string, info *provider.StorageItem) bool {
	if r.URL.Query().Get(`thumbnail`) == `true` && provider.ImageExtensions[path.Ext(info.Name)] {
		return a.thumbnailApp.ServeIfPresent(w, r, pathname)
	}

	return false
}

// GetWithMessage output content with given message
func (a *App) GetWithMessage(w http.ResponseWriter, r *http.Request, request *provider.Request, message *provider.Message) {
	pathname := provider.GetPathname(request, nil)

	info, err := a.storage.Info(pathname)
	if err != nil {
		if !provider.IsNotExist(err) {
			a.renderer.Error(w, http.StatusNotFound, fmt.Errorf(`Requested path does not exist: %s`, request.Path))
		} else {
			a.renderer.Error(w, http.StatusInternalServerError, fmt.Errorf(`Error while reading %s: %s`, request.Path, err))
		}
		return
	}

	if !info.IsDir {
		a.storage.Serve(w, r, pathname)
		return
	}

	if !strings.HasSuffix(r.URL.Path, `/`) {
		http.Redirect(w, r, fmt.Sprintf(`%s/`, r.URL.Path), http.StatusPermanentRedirect)
		return
	}

	if !a.checkAndServeThumbnail(w, r, pathname, info) {
		a.List(w, request, pathname, r.URL.Query().Get(`d`), message)
	}
}

// Get output content
func (a *App) Get(w http.ResponseWriter, r *http.Request, config *provider.Request) {
	var message *provider.Message

	if messageContent := r.URL.Query().Get(`message`); messageContent != `` {
		message = &provider.Message{
			Level:   r.URL.Query().Get(`messageLevel`),
			Content: messageContent,
		}
	}

	a.GetWithMessage(w, r, config, message)
}
