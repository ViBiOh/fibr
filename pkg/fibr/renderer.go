package fibr

import (
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"strings"

	"github.com/ViBiOh/auth/v2/pkg/middleware"
	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/httputils/v4/pkg/model"
	"github.com/ViBiOh/httputils/v4/pkg/query"
	"github.com/ViBiOh/httputils/v4/pkg/renderer"
	"github.com/ViBiOh/httputils/v4/pkg/telemetry"
)

var FuncMap = template.FuncMap{
	"rebuildPaths": func(parts []string, index int) string {
		return fmt.Sprintf("/%s/", strings.Join(parts[:index+1], "/"))
	},
	"join": func(arr []string, separator string) string {
		return strings.Join(arr, separator)
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

func (s Service) TemplateFunc(w http.ResponseWriter, r *http.Request) (renderer.Page, error) {
	if !isMethodAllowed(r) {
		return renderer.Page{}, model.WrapMethodNotAllowed(errors.New("you lack of method for calling me"))
	}

	ctx := r.Context()

	if r.URL.Path == "/sitemap.xml" {
		telemetry.SetRouteTag(ctx, "/sitemap.xml")
		return renderer.NewPage("sitemap", http.StatusOK, nil), nil
	}

	if query.GetBool(r, "redirect") {
		params := r.URL.Query()
		params.Del("redirect")

		s.renderer.Redirect(w, r, fmt.Sprintf("%s?%s", r.URL.Path, params.Encode()), renderer.ParseMessage(r))
		return renderer.Page{}, nil
	}

	request, err := s.parseRequest(r)
	if err != nil {
		if errors.Is(err, model.ErrUnauthorized) {
			w.Header().Add("WWW-Authenticate", `Basic realm="fibr" charset="UTF-8"`)
		}

		content := map[string]any{"Request": request}

		if errors.Is(err, middleware.ErrEmptyAuth) {
			s.renderer.Error(w, r, content, err, renderer.WithNoLog())

			return renderer.Page{}, nil
		}

		return renderer.NewPage("", 0, content), err
	}

	switch r.Method {
	case http.MethodGet:
		return s.crud.Get(w, r, request)
	case http.MethodPost:
		s.crud.Post(w, r, request)
	case http.MethodPut:
		s.crud.Create(w, r, request)
	case http.MethodPatch:
		s.crud.Rename(w, r, request)
	case http.MethodDelete:
		s.crud.Delete(w, r, request)
	}

	return renderer.Page{}, nil
}
