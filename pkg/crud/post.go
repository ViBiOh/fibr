package crud

import (
	"context"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"

	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/httputils/v4/pkg/model"
	"github.com/ViBiOh/httputils/v4/pkg/query"
	"github.com/ViBiOh/httputils/v4/pkg/renderer"
	"github.com/ViBiOh/httputils/v4/pkg/telemetry"
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
			return nil, nil, fmt.Errorf("error while reader multipart: %w", err)
		}

		formName := part.FormName()
		switch formName {
		case "file":
			return values, part, nil

		default:
			value, err := io.ReadAll(part)
			if err != nil {
				return nil, nil, fmt.Errorf("read form value `%s`: %w", formName, err)
			}

			values[formName] = string(value)
		}
	}
}

func (s Service) Post(w http.ResponseWriter, r *http.Request, request provider.Request) {
	contentType := r.Header.Get("Content-Type")

	if query.GetBool(r, "thumbnail") {
		s.thumbnail.Save(w, r, request)
		return
	}

	if contentType == "application/x-www-form-urlencoded" {
		s.handleFormURLEncoded(w, r, request)
		return
	}

	if strings.HasPrefix(contentType, "multipart/form-data") {
		s.handleMultipart(w, r, request)
		return
	}

	s.error(w, r, request, model.WrapMethodNotAllowed(fmt.Errorf("unknown content-type %s", contentType)))
}

func (s Service) handleFormURLEncoded(w http.ResponseWriter, r *http.Request, request provider.Request) {
	ctx := r.Context()
	method := r.FormValue("method")

	switch r.FormValue("type") {
	case "share":
		telemetry.SetRouteTag(ctx, "/share")
		s.handlePostShare(w, r, request, method)

	case "webhook":
		telemetry.SetRouteTag(ctx, "/webhook")
		s.handlePostWebhook(w, r, request, method)

	case "description":
		telemetry.SetRouteTag(ctx, "/description")
		s.handlePostDescription(w, r, request)

	default:
		s.handlePost(w, r, request, method)
	}
}

func (s Service) handleMultipart(w http.ResponseWriter, r *http.Request, request provider.Request) {
	ctx := r.Context()

	if !request.CanEdit {
		s.error(w, r, request, model.WrapForbidden(ErrNotAuthorized))
		return
	}

	values, file, err := parseMultipart(r)
	if err != nil {
		s.error(w, r, request, model.WrapInternal(fmt.Errorf("parse multipart request: %w", err)))
		return
	}

	if values["method"] != http.MethodPost {
		s.error(w, r, request, model.WrapMethodNotAllowed(fmt.Errorf("unknown method `%s` for multipart", values["method"])))
		return
	}

	if len(r.Header.Get("X-Chunk-Upload")) != 0 {
		if chunkNumber := r.Header.Get("X-Chunk-Number"); len(chunkNumber) != 0 {
			chunkNumberValue, err := strconv.ParseUint(chunkNumber, 10, 64)
			if err != nil {
				s.error(w, r, request, model.WrapInvalid(fmt.Errorf("parse chunk number: %w", err)))
			}

			chunkNumber = fmt.Sprintf("%010d", chunkNumberValue)

			telemetry.SetRouteTag(ctx, "/chunk")
			s.uploadChunk(w, r, request, values["filename"], chunkNumber, file)
		} else {
			telemetry.SetRouteTag(ctx, "/merge")
			s.mergeChunk(w, r, request, values)
		}
	} else {
		telemetry.SetRouteTag(ctx, "/upload")
		s.upload(w, r, request, values, file)
	}
}

func (s Service) handlePostShare(w http.ResponseWriter, r *http.Request, request provider.Request, method string) {
	if !request.CanShare {
		s.error(w, r, request, model.WrapForbidden(ErrNotAuthorized))
		return
	}

	switch method {
	case http.MethodPost:
		s.createShare(w, r, request)
	case http.MethodDelete:
		s.deleteShare(w, r, request)
	default:
		s.error(w, r, request, model.WrapMethodNotAllowed(fmt.Errorf("unknown share method `%s` for %s", method, r.URL.Path)))
	}
}

func (s Service) handlePostWebhook(w http.ResponseWriter, r *http.Request, request provider.Request, method string) {
	if !request.CanWebhook {
		s.error(w, r, request, model.WrapForbidden(ErrNotAuthorized))
		return
	}

	switch method {
	case http.MethodPost:
		s.createWebhook(w, r, request)
	case http.MethodDelete:
		s.deleteWebhook(w, r, request)
	default:
		s.error(w, r, request, model.WrapMethodNotAllowed(fmt.Errorf("unknown webhook method `%s` for %s", method, r.URL.Path)))
	}
}

func (s Service) handlePostDescription(w http.ResponseWriter, r *http.Request, request provider.Request) {
	if !request.CanEdit {
		s.error(w, r, request, model.WrapForbidden(ErrNotAuthorized))
		return
	}

	name, err := checkFormName(r, "name")
	if err != nil && !errors.Is(err, ErrEmptyName) {
		s.error(w, r, request, err)
		return
	}

	ctx := r.Context()

	item, err := s.storage.Stat(ctx, request.SubPath(name))
	if err != nil {
		s.error(w, r, request, err)
		return
	}

	description := r.FormValue("description")

	if _, err = s.metadata.Update(ctx, item, provider.ReplaceDescription(description)); err != nil {
		s.error(w, r, request, err)
		return
	}

	go s.pushEvent(context.WithoutCancel(ctx), provider.NewDescriptionEvent(ctx, item, s.bestSharePath(item.Pathname), description, s.renderer))

	s.renderer.Redirect(w, r, fmt.Sprintf("?d=%s#%s", request.Display, item.ID), renderer.NewSuccessMessage("Description successfully edited"))
}

func (s Service) handlePost(w http.ResponseWriter, r *http.Request, request provider.Request, method string) {
	if !request.CanEdit {
		s.error(w, r, request, model.WrapForbidden(ErrNotAuthorized))
		return
	}

	switch method {
	case http.MethodPatch:
		s.Rename(w, r, request)
	case http.MethodPut:
		switch putType := r.FormValue("type"); putType {
		case "folder":
			s.Create(w, r, request)
		case "saved-search":
			s.CreateSavedSearch(w, r, request)
		default:
			s.error(w, r, request, model.WrapInvalid(fmt.Errorf("unknown type `%s`", putType)))
		}
	case http.MethodDelete:
		switch putType := r.FormValue("type"); putType {
		case "file":
			s.Delete(w, r, request)
		case "saved-search":
			s.DeleteSavedSearch(w, r, request)
		default:
			s.error(w, r, request, model.WrapInvalid(fmt.Errorf("unknown type `%s`", putType)))
		}
	case http.MethodTrace:
		s.regenerate(w, r, request)
	default:
		s.error(w, r, request, model.WrapMethodNotAllowed(fmt.Errorf("unknown method `%s` for %s", method, r.URL.Path)))
	}
}
