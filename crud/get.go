package crud

import (
	"errors"
	"io/ioutil"
	"net/http"
	"path"
	"strings"

	"github.com/ViBiOh/fibr/provider"
	"github.com/ViBiOh/fibr/utils"
)

// CheckAndServeSEO check if filename match SEO and serve it, or not
func (a *App) CheckAndServeSEO(w http.ResponseWriter, r *http.Request) bool {
	if r.URL.Path == `/robots.txt` {
		http.ServeFile(w, r, path.Join(`web/static`, r.URL.Path))
		return true
	} else if r.URL.Path == `/sitemap.xml` {
		a.renderer.Sitemap(w)
		return true
	}

	return false
}

// GetDir render directory web view of given dirPath
func (a *App) GetDir(w http.ResponseWriter, config *provider.RequestConfig, filename string, message *provider.Message) {
	files, err := ioutil.ReadDir(filename)
	if err != nil {
		a.renderer.Error(w, http.StatusInternalServerError, err)
		return
	}

	config.Path = strings.TrimPrefix(filename, config.Root)

	a.renderer.Directory(w, config, files, message)
}

// Get write given path from filesystem
func (a *App) Get(w http.ResponseWriter, r *http.Request, config *provider.RequestConfig) {
	filename, info := utils.GetPathInfo(config.Root, config.Path)

	if info == nil {
		if !a.CheckAndServeSEO(w, r) {
			a.renderer.Error(w, http.StatusNotFound, errors.New(`Requested path does not exist`))
		}
	} else if info.IsDir() {
		a.GetDir(w, config, filename, nil)
	} else {
		http.ServeFile(w, r, filename)
	}
}
