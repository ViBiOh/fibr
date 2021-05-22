package renderer

import (
	"html/template"
	"net/http"

	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/httputils/v4/pkg/renderer"
)

// App of package
type App interface {
	Directory(http.ResponseWriter, provider.Request, map[string]interface{}, renderer.Message)
	File(http.ResponseWriter, provider.Request, map[string]interface{}, renderer.Message)
	Error(http.ResponseWriter, provider.Request, *provider.Error)
	Sitemap(http.ResponseWriter)
	SVG(http.ResponseWriter, string, string)
	PublicURL() string
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
