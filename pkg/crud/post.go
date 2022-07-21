package crud

import (
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"

	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/httputils/v4/pkg/model"
	"github.com/ViBiOh/httputils/v4/pkg/renderer"
)

func parseMultipart(r *http.Request) (map[string]string, *multipart.Part, error) {
	reader, err := r.MultipartReader()
	if err != nil {
		return nil, nil, err
	}

	var filePart *multipart.Part
	values := make(map[string]string)

	for {
		part, err := reader.NextPart()
		if err == io.EOF {
			return values, filePart, nil
		}

		if err != nil {
			return nil, nil, fmt.Errorf("error while reader multipart: %s", err)
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
func (a App) Post(w http.ResponseWriter, r *http.Request, request provider.Request) {
	contentType := r.Header.Get("Content-Type")

	if contentType == "application/x-www-form-urlencoded" {
		a.handleFormURLEncoded(w, r, request)
		return
	}

	if strings.HasPrefix(contentType, "multipart/form-data") {
		a.handleMultipart(w, r, request)
		return
	}

	a.error(w, r, request, model.WrapMethodNotAllowed(fmt.Errorf("unknown content-type %s", contentType)))
}

func (a App) handleFormURLEncoded(w http.ResponseWriter, r *http.Request, request provider.Request) {
	method := r.FormValue("method")

	switch r.FormValue("type") {
	case "share":
		a.handlePostShare(w, r, request, method)
	case "webhook":
		a.handlePostWebhook(w, r, request, method)
	case "description":
		a.handlePostDescription(w, r, request, method)
	default:
		a.handlePost(w, r, request, method)
	}
}

func (a App) handleMultipart(w http.ResponseWriter, r *http.Request, request provider.Request) {
	if !request.CanEdit {
		a.error(w, r, request, model.WrapForbidden(ErrNotAuthorized))
		return
	}

	values, file, err := parseMultipart(r)
	if err != nil {
		a.error(w, r, request, model.WrapInternal(fmt.Errorf("unable to parse multipart request: %s", err)))
		return
	}

	if values["method"] != http.MethodPost {
		a.error(w, r, request, model.WrapMethodNotAllowed(fmt.Errorf("unknown method `%s` for multipart", values["method"])))
		return
	}

	if len(r.Header.Get("X-Chunk-Upload")) != 0 {
		if chunkNumber := r.Header.Get("X-Chunk-Number"); len(chunkNumber) != 0 {
			chunkNumberValue, err := strconv.ParseUint(chunkNumber, 10, 64)
			if err != nil {
				a.error(w, r, request, model.WrapInvalid(fmt.Errorf("unable to parse chunk number: %s", err)))
			}

			chunkNumber = fmt.Sprintf("%10d", chunkNumberValue)

			a.uploadChunk(w, r, request, values["filename"], chunkNumber, file)
		} else {
			a.mergeChunk(w, r, request, values)
		}
	} else {
		a.upload(w, r, request, values, file)
	}
}

func (a App) handlePostShare(w http.ResponseWriter, r *http.Request, request provider.Request, method string) {
	switch method {
	case http.MethodPost:
		a.createShare(w, r, request)
	case http.MethodDelete:
		a.deleteShare(w, r, request)
	default:
		a.error(w, r, request, model.WrapMethodNotAllowed(fmt.Errorf("unknown share method `%s` for %s", method, r.URL.Path)))
	}
}

func (a App) handlePostWebhook(w http.ResponseWriter, r *http.Request, request provider.Request, method string) {
	switch method {
	case http.MethodPost:
		a.createWebhook(w, r, request)
	case http.MethodDelete:
		a.deleteWebhook(w, r, request)
	default:
		a.error(w, r, request, model.WrapMethodNotAllowed(fmt.Errorf("unknown webhook method `%s` for %s", method, r.URL.Path)))
	}
}

func (a App) handlePostDescription(w http.ResponseWriter, r *http.Request, request provider.Request, method string) {
	name, err := checkFormName(r, "name")
	if err != nil && !errors.Is(err, ErrEmptyName) {
		a.error(w, r, request, err)
		return
	}

	ctx := r.Context()

	item, err := a.storageApp.Info(ctx, request.SubPath(name))
	if err != nil {
		a.error(w, r, request, err)
		return
	}

	exif, err := a.exifApp.GetExifFor(ctx, item)
	if err != nil {
		a.error(w, r, request, err)
		return
	}

	exif.Description = r.FormValue("description")

	if err = a.exifApp.SaveExifFor(ctx, item, exif); err != nil {
		a.error(w, r, request, err)
		return
	}

	go a.notify(provider.NewDescriptionEvent(item, a.bestSharePath(item.Pathname), exif.Description, a.rendererApp))

	a.rendererApp.Redirect(w, r, fmt.Sprintf("?d=%s#%s", request.Display, item.ID), renderer.NewSuccessMessage("Description successfully edited"))
}

func (a App) handlePost(w http.ResponseWriter, r *http.Request, request provider.Request, method string) {
	switch method {
	case http.MethodPatch:
		a.Rename(w, r, request)
	case http.MethodPut:
		a.Create(w, r, request)
	case http.MethodDelete:
		a.Delete(w, r, request)
	case http.MethodTrace:
		a.regenerate(w, r, request)
	default:
		a.error(w, r, request, model.WrapMethodNotAllowed(fmt.Errorf("unknown method `%s` for %s", method, r.URL.Path)))
	}
}
