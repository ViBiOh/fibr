package ui

import (
	"flag"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"path"
	"regexp"
	"strings"

	"github.com/ViBiOh/fibr/provider"
	"github.com/ViBiOh/fibr/utils"
	"github.com/ViBiOh/httputils"
	"github.com/ViBiOh/httputils/tools"
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
	publicURL := *config[`publicURL`]
	logoutURL := regexp.MustCompile(`(https?://)(.*)`).ReplaceAllString(publicURL, `${1}nobody:nogroup@${2}`)

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
				extension := strings.ToLower(path.Ext(file.Name()))

				switch {
				case provider.ArchiveExtensions[extension]:
					return `-archive`
				case provider.AudioExtensions[extension]:
					return `-audio`
				case provider.CodeExtensions[extension]:
					return `-code`
				case provider.ExcelExtensions[extension]:
					return `-excel`
				case provider.ImageExtensions[extension]:
					return `-image`
				case provider.PdfExtensions[extension]:
					return `-pdf`
				case provider.VideoExtensions[extension]:
					return `-video`
				case provider.WordExtensions[extension]:
					return `-word`
				default:
					return ``
				}
			},
			`isImage`: func(file os.FileInfo) bool {
				return provider.ImageExtensions[path.Ext(file.Name())]
			},
			`hasThumbnail`: func(root, path string, file os.FileInfo) bool {
				_, info := utils.GetPathInfo(root, provider.MetadataDirectoryName, path, file.Name())
				return info != nil
			},
		}).ParseGlob(`./web/*.gohtml`)),

		base: map[string]interface{}{
			`Display`: ``,
			`Config`: map[string]interface{}{
				`PublicURL`: publicURL,
				`LogoutURL`: logoutURL,
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

	if err := httputils.WriteHTMLTemplate(a.tpl.Lookup(`error`), w, errorContent, status); err != nil {
		httputils.InternalServerError(w, err)
	}

	log.Printf(`[error] %v`, err)
}

// Sitemap render sitemap.xml
func (a *App) Sitemap(w http.ResponseWriter) {
	if err := httputils.WriteXMLTemplate(a.tpl.Lookup(`sitemap`), w, a.base, http.StatusOK); err != nil {
		httputils.InternalServerError(w, err)
	}
}

// Directory render directory listing
func (a *App) Directory(w http.ResponseWriter, config *provider.RequestConfig, content map[string]interface{}, display string, message *provider.Message) {
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

	paths := strings.Split(currentPath, `/`)
	if paths[0] == `` {
		paths = nil
	}
	pageContent[`RootName`] = path.Base(config.Root)
	pageContent[`Root`] = config.Root
	pageContent[`Path`] = config.Path
	pageContent[`Paths`] = paths
	pageContent[`CanEdit`] = config.CanEdit
	pageContent[`CanShare`] = config.CanShare

	pageContent[`Display`] = `list`
	if display != `` {
		pageContent[`Display`] = display
	}

	pageContent[`PathPrefix`] = config.PathPrefix
	if config.PathPrefix != `` {
		pageContent[`PathPrefix`] = fmt.Sprintf(`%s/`, config.PathPrefix)
	}

	for key, value := range content {
		pageContent[key] = value
	}

	if err := httputils.WriteHTMLTemplate(a.tpl.Lookup(`files`), w, pageContent, http.StatusOK); err != nil {
		a.Error(w, http.StatusInternalServerError, err)
	}
}
