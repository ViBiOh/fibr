package crud

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
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

// GetDir render directory web view of given dirPath
func (a *App) GetDir(w http.ResponseWriter, request *provider.Request, filename string, display string, message *provider.Message) {
	files, err := ioutil.ReadDir(filename)
	if err != nil {
		a.renderer.Error(w, http.StatusInternalServerError, err)
		return
	}

	paths := strings.Split(strings.Trim(request.Path, `/`), `/`)
	if len(paths) == 1 && paths[0] == `` {
		paths = nil
	}

	content := map[string]interface{}{
		`Paths`: paths,
		`Files`: files,
	}

	if request.CanShare {
		content[`Shares`] = a.metadatas
	}

	a.renderer.Directory(w, request, content, display, message)
}

// GetWithMessage output content with given message
func (a *App) GetWithMessage(w http.ResponseWriter, r *http.Request, request *provider.Request, message *provider.Message) {
	filename, info := a.getFileinfo(request, nil)

	if info == nil {
		a.renderer.Error(w, http.StatusNotFound, fmt.Errorf(`Requested path does not exist: %s`, request.Path))
		return
	}

	if info.IsDir() {
		if !strings.HasSuffix(r.URL.Path, `/`) {
			http.Redirect(w, r, fmt.Sprintf(`%s/`, r.URL.Path), http.StatusPermanentRedirect)
		}

		a.GetDir(w, request, filename, r.URL.Query().Get(`d`), message)
		return
	}

	if params, err := url.ParseQuery(r.URL.RawQuery); err == nil {
		if _, ok := params[`thumbnail`]; ok && provider.ImageExtensions[path.Ext(info.Name())] {
			if tnFilename, tnInfo := a.getMetadataFileinfo(request, nil); tnInfo != nil {
				http.ServeFile(w, r, tnFilename)
				return
			}
		}
	}

	http.ServeFile(w, r, filename)
}

// Get output content
func (a *App) Get(w http.ResponseWriter, r *http.Request, config *provider.Request) {
	a.GetWithMessage(w, r, config, nil)
}
