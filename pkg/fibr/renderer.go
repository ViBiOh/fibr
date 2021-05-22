package fibr

import (
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"path"

	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/fibr/pkg/thumbnail"
	"github.com/ViBiOh/httputils/v4/pkg/model"
	"github.com/ViBiOh/httputils/v4/pkg/query"
	"github.com/ViBiOh/httputils/v4/pkg/renderer"
)

// FuncMap is the map of function available in templates
func FuncMap(thumbnailApp thumbnail.App) template.FuncMap {
	return template.FuncMap{
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
	}
}

func (a app) TemplateFunc(w http.ResponseWriter, r *http.Request) (string, int, map[string]interface{}, error) {
	if !isMethodAllowed(r) {
		return "", 0, nil, model.WrapMethodNotAllowed(errors.New("you lack of method for calling me"))
	}

	if r.URL.Path == "/sitemap.xml" {
		return "sitemap", http.StatusOK, nil, nil
	}

	if query.GetBool(r, "redirect") {
		a.rendererApp.Redirect(w, r, r.URL.Path, renderer.ParseMessage(r))
		return "", 0, nil, nil
	}

	request, err := a.parseRequest(r)
	if err != nil {
		if errors.Is(err, model.ErrUnauthorized) {
			fmt.Println("kiki")
			w.Header().Add("WWW-Authenticate", `Basic realm="fibr" charset="UTF-8"`)
		}
		return "", 0, nil, err
	}

	if r.Method == http.MethodGet {
		return a.crudApp.Get(w, r, request)
	}

	switch r.Method {
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
