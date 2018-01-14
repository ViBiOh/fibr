package crud

import (
	"errors"
	"io/ioutil"
	"net/http"
	"path"
	"strings"

	"github.com/ViBiOh/fibr/ui"
	"github.com/ViBiOh/fibr/utils"
)

// CheckAndServeSEO check if filename match SEO and serve it, or not
func (a *App) CheckAndServeSEO(w http.ResponseWriter, r *http.Request) bool {
	if r.URL.Path == `/robots.txt` {
		http.ServeFile(w, r, path.Join(`web/static`, r.URL.Path))
		return true
	} else if r.URL.Path == `/sitemap.xml` {
		a.uiApp.Sitemap(w)
		return true
	}

	return false
}

// GetDir render directory web view of given dirPath
func (a *App) GetDir(w http.ResponseWriter, url string, rootDirectory string, pathDirectory string, message *ui.Message) {
	files, err := ioutil.ReadDir(pathDirectory)
	if err != nil {
		a.uiApp.Error(w, http.StatusInternalServerError, err)
		return
	}

	a.uiApp.Directory(w, url, path.Base(rootDirectory), strings.TrimPrefix(pathDirectory, rootDirectory), files, message)
}

// Get write given path from filesystem
func (a *App) Get(w http.ResponseWriter, r *http.Request, rootDirectory string) {
	filename, info := utils.GetPathInfo(rootDirectory, r.URL.Path)

	if info == nil {
		if !a.CheckAndServeSEO(w, r) {
			a.uiApp.Error(w, http.StatusNotFound, errors.New(`Requested path does not exist`))
		}
	} else if info.IsDir() {
		a.GetDir(w, r.URL.Path, rootDirectory, filename, nil)
	} else {
		http.ServeFile(w, r, filename)
	}
}
