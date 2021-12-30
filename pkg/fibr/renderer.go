package fibr

import (
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"path"
	"strings"

	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/fibr/pkg/thumbnail"
	"github.com/ViBiOh/httputils/v4/pkg/model"
	"github.com/ViBiOh/httputils/v4/pkg/query"
	"github.com/ViBiOh/httputils/v4/pkg/renderer"
)

// FuncMap is the map of function available in templates
func FuncMap(thumbnailApp thumbnail.App) template.FuncMap {
	return template.FuncMap{
		"rebuildPaths": func(parts []string, index int) string {
			return fmt.Sprintf("/%s/", path.Join(parts[:index+1]...))
		},
		"js": func(content string) template.JS {
			return template.JS(content)
		},
		"join": func(arr []string) string {
			return strings.Join(arr, ",")
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
			return thumbnailApp.CanHaveThumbnail(item.StorageItem) && thumbnailApp.HasThumbnail(item.StorageItem)
		},
	}
}

// TemplateFunc for rendering GUI
func (a App) TemplateFunc(w http.ResponseWriter, r *http.Request) (string, int, map[string]interface{}, error) {
	if !isMethodAllowed(r) {
		return "", 0, nil, model.WrapMethodNotAllowed(errors.New("you lack of method for calling me"))
	}

	if r.URL.Path == "/sitemap.xml" {
		return "sitemap", http.StatusOK, nil, nil
	}

	if query.GetBool(r, "redirect") {
		params := r.URL.Query()
		params.Del("redirect")

		a.rendererApp.Redirect(w, r, fmt.Sprintf("%s?%s", r.URL.Path, params.Encode()), renderer.ParseMessage(r))
		return "", 0, nil, nil
	}

	request, err := a.parseRequest(r)
	if err != nil {
		if errors.Is(err, model.ErrUnauthorized) {
			w.Header().Add("WWW-Authenticate", `Basic realm="fibr" charset="UTF-8"`)
		}
		return "", 0, nil, err
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

	return "", 0, nil, nil
}
