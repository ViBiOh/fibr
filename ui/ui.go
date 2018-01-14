package ui

import (
	"flag"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"path"
	"strings"

	"github.com/ViBiOh/fibr/provider"
	"github.com/ViBiOh/httputils"
	"github.com/ViBiOh/httputils/tools"
)

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

// App for rendering UI
type App struct {
	base map[string]interface{}
	tpl  *template.Template
}

// Flags add flags for given prefix
func Flags(prefix string) map[string]*string {
	return map[string]*string{
		`publicURL`: flag.String(tools.ToCamel(fmt.Sprintf(`%sPublicURL`, prefix)), `https://fibr.vibioh.fr`, `Public Server URL`),
		`staticURL`: flag.String(tools.ToCamel(fmt.Sprintf(`%sStaticURL`, prefix)), `https://fibr-static.vibioh.fr`, `Static Server URL`),
		`version`:   flag.String(tools.ToCamel(fmt.Sprintf(`%sVersion`, prefix)), ``, `Version (used mainly as a cache-buster)`),
	}
}

// NewApp create ui from given config
func NewApp(config map[string]*string, authURL string) *App {
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

		base: map[string]interface{}{
			`Config`: map[string]interface{}{
				`PublicURL`: *config[`publicURL`],
				`StaticURL`: *config[`staticURL`],
				`AuthURL`:   authURL,
				`Version`:   *config[`version`],
			},
			`Seo`: map[string]interface{}{
				`Title`:       `fibr`,
				`Description`: fmt.Sprintf(`FIle BRowser`),
				`URL`:         `/`,
				`Img`:         path.Join(*config[`staticURL`], `/favicon/android-chrome-512x512.png`),
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

	w.Header().Add(`Cache-Control`, `no-cache`)
	if err := httputils.WriteHTMLTemplate(a.tpl.Lookup(`error`), w, errorContent, status); err != nil {
		httputils.InternalServerError(w, err)
	}

	log.Print(err)
}

// Login render login page
func (a *App) Login(w http.ResponseWriter, message *provider.Message) {
	loginContent := cloneContent(a.base)
	if message != nil {
		loginContent[`Message`] = message
	}

	w.Header().Add(`Cache-Control`, `no-cache`)
	if err := httputils.WriteHTMLTemplate(a.tpl.Lookup(`login`), w, loginContent, http.StatusOK); err != nil {
		a.Error(w, http.StatusInternalServerError, err)
	}
}

// Sitemap render sitemap.xml
func (a *App) Sitemap(w http.ResponseWriter) {
	if err := httputils.WriteXMLTemplate(a.tpl.Lookup(`sitemap`), w, a.base, http.StatusOK); err != nil {
		httputils.InternalServerError(w, err)
	}
}

// Directory render directory listing
func (a *App) Directory(w http.ResponseWriter, config *provider.RequestConfig, files []os.FileInfo, message *provider.Message) {
	pageContent := cloneContent(a.base)
	if message != nil {
		pageContent[`Message`] = message
	}

	currentPath := strings.Trim(strings.TrimPrefix(config.Path, config.Root), `/`)

	seo := a.base[`Seo`].(map[string]interface{})
	pageContent[`Seo`] = map[string]interface{}{
		`Title`:       fmt.Sprintf(`fibr - %s`, path.Join(path.Base(config.Root), currentPath)),
		`Description`: fmt.Sprintf(`FIle BRowser of directory %s`, path.Join(path.Base(config.Root), currentPath)),
		`URL`:         config.URL,
		`Img`:         seo[`Img`],
		`ImgHeight`:   seo[`ImgHeight`],
		`ImgWidth`:    seo[`ImgWidth`],
	}

	pageContent[`Files`] = files

	paths := strings.Split(currentPath, `/`)
	if paths[0] == `` {
		paths = nil
	}
	pageContent[`RootName`] = path.Base(config.Root)
	pageContent[`Paths`] = paths
	pageContent[`PathPrefix`] = fmt.Sprintf(`%s/`, config.PathPrefix)
	pageContent[`CanEdit`] = config.CanEdit

	w.Header().Add(`Cache-Control`, `no-cache`)
	if err := httputils.WriteHTMLTemplate(a.tpl.Lookup(`files`), w, pageContent, http.StatusOK); err != nil {
		a.Error(w, http.StatusInternalServerError, err)
	}
}
