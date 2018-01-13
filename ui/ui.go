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

// App for rendering UI
type App struct {
	rootDir string
	base    map[string]interface{}
	tpl     *template.Template
}

var (
	archiveExtension = map[string]bool{`.zip`: true, `.tar`: true, `.gz`: true, `.rar`: true}
	audioExtension   = map[string]bool{`.mp3`: true}
	codeExtension    = map[string]bool{`.html`: true, `.css`: true, `.js`: true, `.jsx`: true, `.json`: true, `.yml`: true, `.yaml`: true, `.toml`: true, `.md`: true, `.go`: true, `.py`: true, `.java`: true, `.xml`: true}
	excelExtension   = map[string]bool{`.xls`: true, `.xlsx`: true, `.xlsm`: true}
	imageExtension   = map[string]bool{`.jpg`: true, `.jpeg`: true, `.png`: true, `.gif`: true, `.svg`: true, `.tiff`: true}
	pdfExtension     = map[string]bool{`.pdf`: true}
	videoExtension   = map[string]bool{`.mp4`: true, `.mov`: true, `.avi`: true}
	wordExtension    = map[string]bool{`.doc`: true, `.docx`: true, `.docm`: true}
)

func cloneContent(content map[string]interface{}) map[string]interface{} {
	clone := make(map[string]interface{})
	for key, value := range content {
		clone[key] = value
	}

	return clone
}

// NewApp create ui from given config
func NewApp(publicURL string, staticURL string, authURL string, version string, rootDirectory string, rootName string) *App {
	return &App{
		tpl: template.Must(template.New(`fibr`).Funcs(template.FuncMap{
			`filename`: func(file os.FileInfo) string {
				if file.IsDir() {
					return fmt.Sprintf(`%s/`, file.Name())
				}
				return file.Name()
			},
			`rebuildPaths`: func(parts []string, index int) string {
				return path.Join(parts[:index+1]...)
			},
			`typeFromExtension`: func(file os.FileInfo) string {
				extension := path.Ext(file.Name())

				switch {
				case archiveExtension[extension]:
					return `-archive`
				case audioExtension[extension]:
					return `-audio`
				case codeExtension[extension]:
					return `-code`
				case excelExtension[extension]:
					return `-excel`
				case imageExtension[extension]:
					return `-image`
				case pdfExtension[extension]:
					return `-pdf`
				case videoExtension[extension]:
					return `-video`
				case wordExtension[extension]:
					return `-word`
				default:
					return ``
				}
			},
		}).ParseGlob(`./web/*.gohtml`)),

		rootDir: rootDirectory,
		base: map[string]interface{}{
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
		},
	}
}

// Error render error page with given status
func (a *App) Error(w http.ResponseWriter, status int, err error) {
	errorContent := cloneContent(a.base)
	errorContent[`Status`] = status
	if err != nil {
		errorContent[`Error`] = err.Error()
	}

	if err := httputils.WriteHTMLTemplate(a.tpl.Lookup(`error`), w, errorContent, status); err != nil {
		httputils.InternalServerError(w, err)
	}
}

// Login render login page
func (a *App) Login(w http.ResponseWriter, message *Message) {
	loginContent := cloneContent(a.base)
	if message != nil {
		loginContent[`Message`] = message
	}

	if err := httputils.WriteHTMLTemplate(a.tpl.Lookup(`login`), w, loginContent, http.StatusOK); err != nil {
		httputils.InternalServerError(w, err)
	}
}

// Sitemap render sitemap.xml
func (a *App) Sitemap(w http.ResponseWriter) {
	if err := httputils.WriteHTMLTemplate(a.tpl.Lookup(`sitemap`), w, a.base, http.StatusOK); err != nil {
		httputils.InternalServerError(w, err)
	}
}

// Directory render directory listing
func (a *App) Directory(w http.ResponseWriter, path string, files []os.FileInfo, message *Message) {
	pageContent := cloneContent(a.base)
	if message != nil {
		pageContent[`Message`] = message
	}

	seo := a.base[`Seo`].(map[string]interface{})
	pageContent[`Seo`] = map[string]interface{}{
		`Title`:       fmt.Sprintf(`fibr - %s`, path),
		`Description`: fmt.Sprintf(`FIle BRowser of directory %s`, path),
		`URL`:         path,
		`Img`:         seo[`Img`],
		`ImgHeight`:   seo[`ImgHeight`],
		`ImgWidth`:    seo[`ImgWidth`],
	}

	pageContent[`Files`] = files

	pathParts := strings.Split(strings.Trim(strings.TrimPrefix(path, a.rootDir), `/`), `/`)
	if pathParts[0] == `` {
		pathParts = nil
	}
	pageContent[`PathParts`] = pathParts

	if err := httputils.WriteHTMLTemplate(a.tpl.Lookup(`files`), w, pageContent, http.StatusOK); err != nil {
		httputils.InternalServerError(w, err)
	}
}
