package renderer

import (
	"embed"
	"flag"
	"fmt"
	"html/template"
	"net/http"
	"path"
	"strings"

	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/fibr/pkg/thumbnail"
	"github.com/ViBiOh/httputils/v4/pkg/flags"
	"github.com/ViBiOh/httputils/v4/pkg/logger"
	"github.com/ViBiOh/httputils/v4/pkg/renderer"
)

//go:embed templates
var filesystem embed.FS

// App of package
type App interface {
	Directory(http.ResponseWriter, provider.Request, map[string]interface{}, renderer.Message)
	File(http.ResponseWriter, provider.Request, map[string]interface{}, renderer.Message)
	Error(http.ResponseWriter, provider.Request, *provider.Error)
	Sitemap(http.ResponseWriter)
	SVG(http.ResponseWriter, string, string)
}

// Config of package
type Config struct {
	publicURL *string
	version   *string
}

type app struct {
	tpl    *template.Template
	config provider.Config
}

// Flags adds flags for configuring package
func Flags(fs *flag.FlagSet, prefix string) Config {
	return Config{
		publicURL: flags.New(prefix, "fibr").Name("PublicURL").Default("https://fibr.vibioh.fr").Label("Public URL").ToString(fs),
		version:   flags.New(prefix, "fibr").Name("Version").Default("").Label("Version (used mainly as a cache-buster)").ToString(fs),
	}
}

// New creates new App from Config
func New(config Config, thumbnailApp thumbnail.App) App {
	tpl, err := template.New("fibr").Funcs(template.FuncMap{
		"asyncImage": func(file provider.RenderItem, version string) map[string]interface{} {
			return map[string]interface{}{
				"File":    file,
				"Version": version,
			}
		},
		"rebuildPaths": func(parts []string, index int) string {
			return path.Join(parts[:index+1]...)
		},
		"iconFromExtension": func(file provider.RenderItem) string {
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
		"hasThumbnail": func(item provider.RenderItem) bool {
			return thumbnail.CanHaveThumbnail(item.StorageItem) && thumbnailApp.HasThumbnail(item.StorageItem)
		},
	}).ParseFS(filesystem, "templates/*.html")
	logger.Fatal(err)

	publicURL := strings.TrimSpace(*config.publicURL)
	imgSize := uint(512)

	return app{
		tpl: tpl,
		config: provider.Config{
			PublicURL: publicURL,
			Version:   *config.version,
			Seo: provider.Seo{
				Title:       "fibr",
				Description: "FIle BRowser",
				Img:         fmt.Sprintf("%s/favicon/android-chrome-%dx%d.png", publicURL, imgSize, imgSize),
				ImgHeight:   imgSize,
				ImgWidth:    imgSize,
			},
		},
	}
}
