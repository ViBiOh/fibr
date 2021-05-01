package crud

import (
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strings"

	"github.com/ViBiOh/fibr/pkg/provider"
)

func parseMultipart(r *http.Request) (map[string]string, *multipart.Part, error) {
	reader, err := r.MultipartReader()
	if err != nil {
		return nil, nil, err
	}

	var filePart *multipart.Part
	values := make(map[string]string, 0)

	for {
		part, err := reader.NextPart()
		if err == io.EOF {
			return values, filePart, nil
		}

		formName := part.FormName()
		switch formName {
		case "file":
			return values, part, nil

		default:
			value, err := io.ReadAll(part)
			if err != nil {
				return nil, nil, fmt.Errorf("unable to read form value `%s`: %s", formName, err)
			}

			values[formName] = string(value)
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
				a.rendererApp.Error(w, request, provider.NewError(http.StatusMethodNotAllowed, fmt.Errorf("unknown share method `%s` for %s", method, r.URL.Path)))
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
				a.rendererApp.Error(w, request, provider.NewError(http.StatusMethodNotAllowed, fmt.Errorf("unknown method `%s` for %s", method, r.URL.Path)))
			}
		}

		return
	}

	if strings.HasPrefix(contentType, "multipart/form-data") {
		values, file, err := parseMultipart(r)
		if err != nil {
			a.rendererApp.Error(w, request, provider.NewError(http.StatusInternalServerError, fmt.Errorf("unable to parse multipart request: %s", err)))
			return
		}

		if values["method"] != http.MethodPost {
			a.rendererApp.Error(w, request, provider.NewError(http.StatusMethodNotAllowed, fmt.Errorf("unknown method `%s` for multipart", values["method"])))
			return
		}

		a.Upload(w, r, request, values, file)
		return
	}

	a.rendererApp.Error(w, request, provider.NewError(http.StatusMethodNotAllowed, fmt.Errorf("unknown content-type %s", contentType)))
}
