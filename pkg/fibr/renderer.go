package fibr

import (
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"strings"

	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/httputils/v4/pkg/model"
	"github.com/ViBiOh/httputils/v4/pkg/query"
	"github.com/ViBiOh/httputils/v4/pkg/renderer"
)

// FuncMap is the map of function available in templates
var FuncMap = template.FuncMap{
	"rebuildPaths": func(parts []string, index int) string {
		return fmt.Sprintf("/%s/", strings.Join(parts[:index+1], "/"))
	},
	"raw": func(content string) template.URL {
		return template.URL(content)
	},
	"splitLines": func(value string) []string {
		return strings.Split(value, "\n")
	},
	"add": func(a, b int) int {
		return a + b
	},
	"contains": func(arr []string, value string) bool {
		for _, item := range arr {
			if item == value {
				return true
			}
		}

		return false
	},
	"iconFromExtension": func(file provider.RenderItem) string {
		switch {
		case provider.ArchiveExtensions[file.Extension]:
			return "file-archive"
		case provider.AudioExtensions[file.Extension]:
			return "file-audio"
		case provider.CodeExtensions[file.Extension]:
			return "file-code"
		case provider.ExcelExtensions[file.Extension]:
			return "file-excel"
		case provider.ImageExtensions[file.Extension]:
			return "file-image"
		case provider.PdfExtensions[file.Extension]:
			return "file-pdf"
		case provider.VideoExtensions[file.Extension] != "":
			return "file-video"
		case provider.WordExtensions[file.Extension]:
			return "file-word"
		default:
			return "file"
		}
	},
}

// TemplateFunc for rendering GUI
func (a App) TemplateFunc(w http.ResponseWriter, r *http.Request) (renderer.Page, error) {
	if !isMethodAllowed(r) {
		return renderer.Page{}, model.WrapMethodNotAllowed(errors.New("you lack of method for calling me"))
	}

	if r.URL.Path == "/sitemap.xml" {
		return renderer.NewPage("sitemap", http.StatusOK, nil), nil
	}

	if query.GetBool(r, "redirect") {
		params := r.URL.Query()
		params.Del("redirect")

		a.rendererApp.Redirect(w, r, fmt.Sprintf("%s?%s", r.URL.Path, params.Encode()), renderer.ParseMessage(r))
		return renderer.Page{}, nil
	}

	request, err := a.parseRequest(r)
	if err != nil {
		if errors.Is(err, model.ErrUnauthorized) {
			w.Header().Add("WWW-Authenticate", `Basic realm="fibr" charset="UTF-8"`)
		}
		return renderer.NewPage("", 0, map[string]any{"Request": request}), err
	}

	switch r.Method {
	case http.MethodGet:
		return a.crudApp.Get(w, r, request)
	case http.MethodPost:
		a.crudApp.Post(w, r, request)
	case http.MethodPut:
		a.crudApp.Create(w, r, request)
	case http.MethodPatch:
		a.crudApp.Rename(w, r, request)
	case http.MethodDelete:
		a.crudApp.Delete(w, r, request)
	}

	return renderer.Page{}, nil
}
