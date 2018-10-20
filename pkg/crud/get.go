package crud

import (
	"fmt"
	"net/http"
	"path"
	"strings"

	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/httputils/pkg/query"
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

func isThumbnail(r *http.Request) bool {
	return query.GetBool(r, `thumbnail`)
}

func (a *App) checkAndServeThumbnail(w http.ResponseWriter, r *http.Request, pathname string, info *provider.StorageItem) bool {
	if isThumbnail(r) && provider.ImageExtensions[info.Extension()] {
		return a.thumbnailApp.ServeIfPresent(w, r, pathname)
	}

	return false
}

// GetWithMessage output content with given message
func (a *App) GetWithMessage(w http.ResponseWriter, r *http.Request, request *provider.Request, message *provider.Message) {
	pathname := provider.GetPathname(request, ``)

	info, err := a.storage.Info(pathname)
	if err != nil {
		if provider.IsNotExist(err) {
			a.renderer.Error(w, http.StatusNotFound, err)
		} else {
			a.renderer.Error(w, http.StatusInternalServerError, err)
		}
		return
	}

	if !info.IsDir {
		if a.checkAndServeThumbnail(w, r, pathname, info) {
			return
		}

		if r.URL.Query().Get(`browser`) == `true` {
			a.Browser(w, request, info, message)
			return
		}

		a.storage.Serve(w, r, pathname)
		return
	}

	if !strings.HasSuffix(r.URL.Path, `/`) {
		http.Redirect(w, r, fmt.Sprintf(`%s/`, r.URL.Path), http.StatusPermanentRedirect)
		return
	}

	if isThumbnail(r) {
		a.thumbnailApp.List(w, r, pathname)
	} else {
		a.List(w, request, r.URL.Query().Get(`d`), message)
	}
}

// Get output content
func (a *App) Get(w http.ResponseWriter, r *http.Request, config *provider.Request) {
	var message *provider.Message

	if messageContent := strings.TrimSpace(r.URL.Query().Get(`message`)); messageContent != `` {
		message = &provider.Message{
			Level:   r.URL.Query().Get(`messageLevel`),
			Content: messageContent,
		}
	}

	a.GetWithMessage(w, r, config, message)
}
