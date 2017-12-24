package ui

import (
	"fmt"
	"html/template"
	"net/http"
	"os"
	"path"
	"strings"

	"github.com/ViBiOh/httputils"
)

// Message rendered to user
type Message struct {
	Level   string
	Content string
}

var rootDir string
var base map[string]interface{}
var tpl *template.Template

// Init initialize ui
func Init(baseTpl *template.Template, publicURL string, staticURL string, authURL string, version string, rootDirectory string, rootName string) {
	tpl = baseTpl

	rootDir = rootDirectory
	base = map[string]interface{}{
		`Config`: map[string]interface{}{
			`PublicURL`: publicURL,
			`StaticURL`: staticURL,
			`AuthURL`:   authURL,
			`Version`:   version,
			`Root`:      rootName,
		},
		`Seo`: map[string]interface{}{
			`Title`:       `fibr`,
			`Description`: fmt.Sprintf(`FIle BRowser`),
			`URL`:         `/`,
			`Img`:         path.Join(staticURL, `/favicon/android-chrome-512x512.png`),
			`ImgHeight`:   512,
			`ImgWidth`:    512,
		},
	}
}

func cloneContent(content map[string]interface{}) map[string]interface{} {
	clone := make(map[string]interface{})
	for key, value := range content {
		clone[key] = value
	}

	return clone
}

// Error render error page with given status
func Error(w http.ResponseWriter, status int, err error) {
	errorContent := cloneContent(base)
	errorContent[`Status`] = status
	if err != nil {
		errorContent[`Error`] = err.Error()
	}

	w.WriteHeader(status)
	if err := httputils.WriteHTMLTemplate(tpl.Lookup(`error`), w, errorContent); err != nil {
		httputils.InternalServerError(w, err)
	}
}

// Login render login page
func Login(w http.ResponseWriter, message *Message) {
	loginContent := cloneContent(base)
	if message != nil {
		loginContent[`Message`] = message
	}

	if err := httputils.WriteHTMLTemplate(tpl.Lookup(`login`), w, loginContent); err != nil {
		httputils.InternalServerError(w, err)
	}
}

// Sitemap render sitemap.xml
func Sitemap(w http.ResponseWriter) {
	if err := httputils.WriteHTMLTemplate(tpl.Lookup(`sitemap`), w, base); err != nil {
		httputils.InternalServerError(w, err)
	}
}

// Directory render directory listing
func Directory(w http.ResponseWriter, path string, files []os.FileInfo, message *Message) {
	pageContent := cloneContent(base)
	if message != nil {
		pageContent[`Message`] = message
	}

	seo := base[`Seo`].(map[string]interface{})
	pageContent[`Seo`] = map[string]interface{}{
		`Title`:       fmt.Sprintf(`fibr - %s`, path),
		`Description`: fmt.Sprintf(`FIle BRowser of directory %s`, path),
		`URL`:         path,
		`Img`:         seo[`Img`],
		`ImgHeight`:   seo[`ImgHeight`],
		`ImgWidth`:    seo[`ImgWidth`],
	}

	pageContent[`Files`] = files

	pathParts := strings.Split(strings.Trim(strings.TrimPrefix(path, rootDir), `/`), `/`)
	if pathParts[0] == `` {
		pathParts = nil
	}
	pageContent[`PathParts`] = pathParts

	if err := httputils.WriteHTMLTemplate(tpl.Lookup(`files`), w, pageContent); err != nil {
		httputils.InternalServerError(w, err)
	}
}
