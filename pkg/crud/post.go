package crud

import (
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strings"

	"github.com/ViBiOh/fibr/pkg/provider"
)

func parseMultipart(r *http.Request) (string, *multipart.Part, error) {
	reader, err := r.MultipartReader()
	if err != nil {
		return "", nil, err
	}

	var (
		method   string
		filePart *multipart.Part
	)

	for {
		part, err := reader.NextPart()
		if err == io.EOF {
			return method, filePart, nil
		}

		switch part.FormName() {
		case "method":
			value, err := io.ReadAll(part)
			if err != nil {
				return "", nil, err
			}

			method = string(value)

		case "file":
			if len(method) != 0 {
				return method, part, nil
			}
			filePart = part
		}
	}
}

// Post handle post from form
func (a *app) Post(w http.ResponseWriter, r *http.Request, request provider.Request) {
	contentType := r.Header.Get("Content-Type")

	if contentType == "application/x-www-form-urlencoded" {
		method := r.FormValue("method")

		if r.FormValue("type") == "share" {
			switch method {
			case http.MethodPost:
				a.CreateShare(w, r, request)
			case http.MethodDelete:
				a.DeleteShare(w, r, request)
			default:
				a.renderer.Error(w, request, provider.NewError(http.StatusMethodNotAllowed, fmt.Errorf("unknown share method `%s` for %s", method, r.URL.Path)))
			}
		} else {
			switch method {
			case http.MethodPatch:
				a.Rename(w, r, request)
			case http.MethodPut:
				a.Create(w, r, request)
			case http.MethodDelete:
				a.Delete(w, r, request)
			default:
				a.renderer.Error(w, request, provider.NewError(http.StatusMethodNotAllowed, fmt.Errorf("unknown method `%s` for %s", method, r.URL.Path)))
			}
		}

		return
	}

	if strings.HasPrefix(contentType, "multipart/form-data") {
		method, file, err := parseMultipart(r)
		if err != nil {
			a.renderer.Error(w, request, provider.NewError(http.StatusInternalServerError, fmt.Errorf("unable to parse multipart request: %s", err)))
			return
		}

		if method != http.MethodPost {
			a.renderer.Error(w, request, provider.NewError(http.StatusMethodNotAllowed, fmt.Errorf("unknown method `%s` for multipart", method)))
			return
		}

		a.Upload(w, r, request, file)
		return
	}

	a.renderer.Error(w, request, provider.NewError(http.StatusMethodNotAllowed, fmt.Errorf("unknown content-type %s", contentType)))
}
