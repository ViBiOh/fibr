package ui

import (
	"flag"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"net/url"
	"path"
	"strings"

	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/fibr/pkg/thumbnail"
	"github.com/ViBiOh/fibr/pkg/utils"
	"github.com/ViBiOh/httputils/pkg/httperror"
	"github.com/ViBiOh/httputils/pkg/httpjson"
	"github.com/ViBiOh/httputils/pkg/templates"
	"github.com/ViBiOh/httputils/pkg/tools"
)

// App for rendering UI
type App struct {
	config       *provider.Config
	tpl          *template.Template
	thumbnailApp *thumbnail.App
}

// Flags add flags for given prefix
func Flags(prefix string) map[string]*string {
	return map[string]*string{
		`publicURL`: flag.String(tools.ToCamel(fmt.Sprintf(`%sPublicURL`, prefix)), `https://fibr.vibioh.fr`, `[fibr] Public URL`),
		`version`:   flag.String(tools.ToCamel(fmt.Sprintf(`%sVersion`, prefix)), ``, `[fibr] Version (used mainly as a cache-buster)`),
	}
}

// NewApp create ui from given config
func NewApp(config map[string]*string, rootName string, thumbnailApp *thumbnail.App) *App {
	tpl := template.New(`fibr`)

	tpl.Funcs(template.FuncMap{
		`urlescape`: func(path string) string {
			return url.PathEscape(path)
		},
		`sha1`: func(file *provider.StorageItem) string {
			return tools.Sha1(file.Name)
		},
		`asyncImage`: func(file *provider.StorageItem, version string) map[string]interface{} {
			return map[string]interface{}{
				`File`:        file,
				`Fingerprint`: template.JS(tools.Sha1(file.Name)),
				`Version`:     version,
			}
		},
		`rebuildPaths`: func(parts []string, index int) string {
			return path.Join(parts[:index+1]...)
		},
		`iconFromExtension`: func(file *provider.StorageItem) string {
			extension := strings.ToLower(path.Ext(file.Name))

			switch {
			case provider.ArchiveExtensions[extension]:
				return `file-archive`
			case provider.AudioExtensions[extension]:
				return `file-audio`
			case provider.CodeExtensions[extension]:
				return `file-code`
			case provider.ExcelExtensions[extension]:
				return `file-excel`
			case provider.ImageExtensions[extension]:
				return `file-image`
			case provider.PdfExtensions[extension]:
				return `file-pdf`
			case provider.VideoExtensions[extension]:
				return `file-video`
			case provider.WordExtensions[extension]:
				return `file-word`
			default:
				return `file`
			}
		},
		`isImage`: func(file *provider.StorageItem) bool {
			return provider.ImageExtensions[path.Ext(file.Name)]
		},
		`extension`: func(file *provider.StorageItem) string {
			return strings.ToLower(strings.TrimPrefix(path.Ext(file.Name), `.`))
		},
		`hasThumbnail`: func(request *provider.Request, file *provider.StorageItem) bool {
			return thumbnailApp.IsExist(provider.GetPathname(request, []byte(file.Name)))
		},
	})

	fibrTemplates, err := utils.ListFilesByExt(`./templates/`, `.gohtml`)
	if err != nil {
		log.Fatalf(`Error while getting templates: %v`, err)
	}

	publicURL := *config[`publicURL`]

	return &App{
		tpl: template.Must(tpl.ParseFiles(fibrTemplates...)),
		config: &provider.Config{
			RootName:  rootName,
			PublicURL: publicURL,
			Version:   *config[`version`],
			Seo: &provider.Seo{
				Title:       `fibr`,
				Description: `FIle BRowser`,
				Img:         fmt.Sprintf(`%s/favicon/android-chrome-512x512.png`, publicURL),
				ImgHeight:   512,
				ImgWidth:    512,
			},
		},
		thumbnailApp: thumbnailApp,
	}
}

func (a *App) createPageConfiguration(request *provider.Request, message *provider.Message, content map[string]interface{}, layout string) *provider.Page {
	return &provider.Page{
		Config:  a.config,
		Request: request,
		Message: message,
		Content: content,
		Layout:  layout,
	}
}

// Error render error page with given status
func (a *App) Error(w http.ResponseWriter, status int, err error) {
	page := &provider.Page{
		Config: a.config,
		Error: &provider.Error{
			Status: status,
		},
	}

	if err := templates.WriteHTMLTemplate(a.tpl.Lookup(`error`), w, page, status); err != nil {
		httperror.InternalServerError(w, err)
	}

	log.Printf(`[error] %v`, err)
}

// Sitemap render sitemap.xml
func (a *App) Sitemap(w http.ResponseWriter) {
	if err := templates.WriteXMLTemplate(a.tpl.Lookup(`sitemap`), w, provider.Page{Config: a.config}, http.StatusOK); err != nil {
		httperror.InternalServerError(w, err)
	}
}

// Directory render directory listing
func (a *App) Directory(w http.ResponseWriter, request *provider.Request, content map[string]interface{}, layout string, message *provider.Message) {
	page := a.createPageConfiguration(request, message, content, layout)

	if page.Layout == `` {
		page.Layout = `grid`
	}

	if request.IsDebug {
		if err := httpjson.ResponseJSON(w, http.StatusOK, page, true); err != nil {
			a.Error(w, http.StatusInternalServerError, err)
		}
		return
	}

	w.Header().Set(`content-language`, `fr`)
	if err := templates.WriteHTMLTemplate(a.tpl.Lookup(`files`), w, page, http.StatusOK); err != nil {
		a.Error(w, http.StatusInternalServerError, err)
	}
}

// File render file detail
func (a *App) File(w http.ResponseWriter, request *provider.Request, content map[string]interface{}, message *provider.Message) {
	page := a.createPageConfiguration(request, message, content, `browser`)

	if request.IsDebug {
		if err := httpjson.ResponseJSON(w, http.StatusOK, page, true); err != nil {
			a.Error(w, http.StatusInternalServerError, err)
		}
		return
	}

	w.Header().Set(`content-language`, `fr`)
	if err := templates.WriteHTMLTemplate(a.tpl.Lookup(`file`), w, page, http.StatusOK); err != nil {
		a.Error(w, http.StatusInternalServerError, err)
	}
}
