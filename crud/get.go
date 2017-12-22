package crud

import (
	"html/template"
	"io/ioutil"
	"net/http"
	"path"

	"github.com/ViBiOh/fibr/ui"
	"github.com/ViBiOh/fibr/utils"
	"github.com/ViBiOh/httputils"
)

// CheckAndServeSEO check if filename match SEO and serve it, or not
func CheckAndServeSEO(w http.ResponseWriter, r *http.Request, tpl *template.Template, content map[string]interface{}) bool {
	if r.URL.Path == `/robots.txt` {
		http.ServeFile(w, r, path.Join(`web/static`, r.URL.Path))
		return true
	} else if r.URL.Path == `/sitemap.xml` {
		if err := httputils.WriteXMLTemplate(tpl.Lookup(`sitemap`), w, content); err != nil {
			httputils.InternalServerError(w, err)
		}
		return true
	}

	return false
}

// Get service given path from filesystem
func Get(w http.ResponseWriter, r *http.Request, directory string, tpl *template.Template, content map[string]interface{}) {
	filename, info := utils.GetPathInfo(directory, r.URL.Path)

	if info == nil {
		if !CheckAndServeSEO(w, r, tpl, content) {
			httputils.NotFound(w)
		}
	} else if info.IsDir() {
		files, err := ioutil.ReadDir(filename)
		if err != nil {
			httputils.InternalServerError(w, err)
			return
		}

		if err := httputils.WriteHTMLTemplate(tpl.Lookup(`files`), w, ui.GeneratePageContent(content, r, info, files)); err != nil {
			httputils.InternalServerError(w, err)
		}
	} else {
		http.ServeFile(w, r, filename)
	}
}
