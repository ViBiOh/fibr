package crud

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"path"
	"strings"

	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/fibr/pkg/utils"
)

// CheckAndServeSEO check if filename match SEO and serve it, or not
func (a *App) CheckAndServeSEO(w http.ResponseWriter, r *http.Request) bool {
	if r.Method != http.MethodGet {
		return false
	}

	if r.URL.Path == `/robots.txt` || strings.HasPrefix(r.URL.Path, `/favicon`) {
		http.ServeFile(w, r, path.Join(`web/static`, r.URL.Path))
		return true
	}

	if r.URL.Path == `/sitemap.xml` {
		a.renderer.Sitemap(w)
		return true
	}

	return false
}

// GetDir render directory web view of given dirPath
func (a *App) GetDir(w http.ResponseWriter, config *provider.RequestConfig, filename string, display string, message *provider.Message) {
	files, err := ioutil.ReadDir(filename)
	if err != nil {
		a.renderer.Error(w, http.StatusInternalServerError, err)
		return
	}

	config.Path = strings.TrimPrefix(filename, path.Join(a.rootDirectory, config.Root))

	content := map[string]interface{}{
		`Files`: files,
	}

	if config.CanShare {
		content[`Shares`] = a.metadatas
	}

	a.renderer.Directory(w, config, content, display, message)
}

// GetWithMessage output content with given message
func (a *App) GetWithMessage(w http.ResponseWriter, r *http.Request, config *provider.RequestConfig, message *provider.Message) {
	filename, info := utils.GetPathInfo(a.rootDirectory, config.Root, config.Path)

	if info == nil {
		a.renderer.Error(w, http.StatusNotFound, fmt.Errorf(`Requested path does not exist: %s`, config.Path))
		return
	}

	if info.IsDir() {
		a.GetDir(w, config, filename, r.URL.Query().Get(`d`), message)
		return
	}

	if params, err := url.ParseQuery(r.URL.RawQuery); err == nil {
		if _, ok := params[`thumbnail`]; ok && provider.ImageExtensions[path.Ext(info.Name())] {
			if tnFilename, tnInfo := utils.GetPathInfo(a.rootDirectory, provider.MetadataDirectoryName, config.Root, config.Path); tnInfo != nil {
				http.ServeFile(w, r, tnFilename)
				return
			}
		}
	}

	http.ServeFile(w, r, filename)
}

// Get output content
func (a *App) Get(w http.ResponseWriter, r *http.Request, config *provider.RequestConfig) {
	a.GetWithMessage(w, r, config, nil)
}
