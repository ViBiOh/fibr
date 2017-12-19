package crud

import (
	"fmt"
	"html/template"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"strings"

	"github.com/ViBiOh/fibr/utils"
	"github.com/ViBiOh/httputils"
)

func generatePageContent(baseContent map[string]interface{}, currentPath string, current os.FileInfo, files []os.FileInfo) map[string]interface{} {
	pathParts := strings.Split(strings.Trim(currentPath, `/`), `/`)
	if pathParts[0] == `` {
		pathParts = nil
	}

	seo := baseContent[`Seo`].(map[string]interface{})
	pageContent := map[string]interface{}{
		`Config`: baseContent[`Config`],
	}

	pageContent[`PathParts`] = pathParts
	pageContent[`Current`] = current
	pageContent[`Files`] = files
	pageContent[`Seo`] = map[string]interface{}{
		`Title`:       fmt.Sprintf(`fibr - %s`, currentPath),
		`Description`: fmt.Sprintf(`FIle BRowser of directory %s on the server`, currentPath),
		`URL`:         currentPath,
		`Img`:         seo[`Img`],
		`ImgHeight`:   seo[`ImgHeight`],
		`ImgWidth`:    seo[`ImgWidth`],
	}

	return pageContent
}

// CheckAndServeSEO check if filename match SEO and serve it, or not
func CheckAndServeSEO(w http.ResponseWriter, r *http.Request, tpl *template.Template, content map[string]interface{}) bool {
	if r.URL.Path == `/robots.txt` {
		http.ServeFile(w, r, path.Join(`web/static`, r.URL.Path))
		return true
	} else if r.URL.Path == `/sitemap.xml` {
		if err := utils.WriteXMLTemplate(tpl, w, `sitemap`, content); err != nil {
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

		if err := utils.WriteHTMLTemplate(tpl, w, `files`, generatePageContent(content, r.URL.Path, info, files)); err != nil {
			httputils.InternalServerError(w, err)
		}
	} else {
		http.ServeFile(w, r, filename)
	}
}
