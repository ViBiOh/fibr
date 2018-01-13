package crud

import (
	"errors"
	"io/ioutil"
	"net/http"
	"path"

	"github.com/ViBiOh/fibr/ui"
	"github.com/ViBiOh/fibr/utils"
)

// CheckAndServeSEO check if filename match SEO and serve it, or not
func CheckAndServeSEO(w http.ResponseWriter, r *http.Request, uiConfig *ui.App) bool {
	if r.URL.Path == `/robots.txt` {
		http.ServeFile(w, r, path.Join(`web/static`, r.URL.Path))
		return true
	} else if r.URL.Path == `/sitemap.xml` {
		uiConfig.Sitemap(w)
		return true
	}

	return false
}

// GetDir render directory web view of given dirPath
func GetDir(w http.ResponseWriter, dirPath string, uiConfig *ui.App, message *ui.Message) {
	files, err := ioutil.ReadDir(dirPath)
	if err != nil {
		uiConfig.Error(w, http.StatusInternalServerError, err)
		return
	}

	uiConfig.Directory(w, dirPath, files, message)
}

// Get write given path from filesystem
func Get(w http.ResponseWriter, r *http.Request, directory string, uiConfig *ui.App) {
	filename, info := utils.GetPathInfo(directory, r.URL.Path)

	if info == nil {
		if !CheckAndServeSEO(w, r, uiConfig) {
			uiConfig.Error(w, http.StatusNotFound, errors.New(`Requested path does not exist`))
		}
	} else if info.IsDir() {
		GetDir(w, filename, uiConfig, nil)
	} else {
		http.ServeFile(w, r, filename)
	}
}
