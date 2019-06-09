package renderer

import (
	"flag"
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"path"
	"strings"

	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/fibr/pkg/thumbnail"
	"github.com/ViBiOh/httputils/pkg/logger"
	"github.com/ViBiOh/httputils/pkg/templates"
	"github.com/ViBiOh/httputils/pkg/tools"
)

// App of package
type App interface {
	Directory(http.ResponseWriter, *provider.Request, map[string]interface{}, string, *provider.Message)
	File(http.ResponseWriter, *provider.Request, map[string]interface{}, *provider.Message)
	Error(http.ResponseWriter, *provider.Error)
	Sitemap(http.ResponseWriter)
	SVG(http.ResponseWriter, string, string)
}

// Config of package
type Config struct {
	publicURL *string
	version   *string
}

type app struct {
	config       *provider.Config
	tpl          *template.Template
	thumbnailApp *thumbnail.App
}

// Flags adds flags for configuring package
func Flags(fs *flag.FlagSet, prefix string) Config {
	return Config{
		publicURL: fs.String(tools.ToCamel(fmt.Sprintf("%sPublicURL", prefix)), "https://fibr.vibioh.fr", "[fibr] Public URL"),
		version:   fs.String(tools.ToCamel(fmt.Sprintf("%sVersion", prefix)), "", "[fibr] Version (used mainly as a cache-buster)"),
	}
}

// New creates new App from Config
func New(config Config, rootName string, thumbnailApp *thumbnail.App) App {
	tpl := template.New("fibr")

	tpl.Funcs(template.FuncMap{
		"urlescape": func(path string) string {
			return url.PathEscape(path)
		},
		"sha1": func(file *provider.StorageItem) string {
			return tools.Sha1(file.Name)
		},
		"asyncImage": func(file *provider.StorageItem, version string) map[string]interface{} {
			return map[string]interface{}{
				"File":        file,
				"Fingerprint": template.JS(tools.Sha1(file.Name)),
				"Version":     version,
			}
		},
		"rebuildPaths": func(parts []string, index int) string {
			return path.Join(parts[:index+1]...)
		},
		"iconFromExtension": func(file *provider.StorageItem) string {
			extension := file.Extension()

			switch {
			case provider.ArchiveExtensions[extension]:
				return "file-archive"
			case provider.AudioExtensions[extension]:
				return "file-audio"
			case provider.CodeExtensions[extension]:
				return "file-code"
			case provider.ExcelExtensions[extension]:
				return "file-excel"
			case provider.ImageExtensions[extension]:
				return "file-image"
			case provider.PdfExtensions[extension]:
				return "file-pdf"
			case provider.VideoExtensions[extension] != "":
				return "file-video"
			case provider.WordExtensions[extension]:
				return "file-word"
			default:
				return "file"
			}
		},
		"hasThumbnail": func(request *provider.Request, file *provider.StorageItem) bool {
			_, ok := thumbnailApp.HasThumbnail(provider.GetPathname(request, file.Name))
			return ok
		},
	})

	fibrTemplates, err := templates.GetTemplates("./templates/", ".html")
	if err != nil {
		logger.Fatal("%#v", err)
	}

	publicURL := strings.TrimSpace(*config.publicURL)

	return app{
		tpl: template.Must(tpl.ParseFiles(fibrTemplates...)),
		config: &provider.Config{
			RootName:  rootName,
			PublicURL: publicURL,
			Version:   *config.version,
			Seo: &provider.Seo{
				Title:       "fibr",
				Description: "FIle BRowser",
				Img:         fmt.Sprintf("%s/favicon/android-chrome-512x512.png", publicURL),
				ImgHeight:   512,
				ImgWidth:    512,
			},
		},
		thumbnailApp: thumbnailApp,
	}
}

// Directory render directory listing
func (a app) Directory(w http.ResponseWriter, request *provider.Request, content map[string]interface{}, layout string, message *provider.Message) {
	builder := &provider.PageBuilder{}
	page := builder.Config(a.config).Request(request).Message(message).Layout(layout).Content(content).Build()

	w.Header().Set("content-language", "en")
	if err := templates.WriteHTMLTemplate(a.tpl.Lookup("files"), w, page, http.StatusOK); err != nil {
		a.Error(w, provider.NewError(http.StatusInternalServerError, err))
		return
	}
}

// File render file detail
func (a app) File(w http.ResponseWriter, request *provider.Request, content map[string]interface{}, message *provider.Message) {
	builder := &provider.PageBuilder{}
	page := builder.Config(a.config).Request(request).Message(message).Layout("browser").Content(content).Build()

	w.Header().Set("content-language", "en")
	if err := templates.WriteHTMLTemplate(a.tpl.Lookup("file"), w, page, http.StatusOK); err != nil {
		a.Error(w, provider.NewError(http.StatusInternalServerError, err))
		return
	}
}
