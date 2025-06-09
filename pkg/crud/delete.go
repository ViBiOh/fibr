package crud

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/httputils/v4/pkg/model"
	"github.com/ViBiOh/httputils/v4/pkg/renderer"
)

func (s *Service) Delete(w http.ResponseWriter, r *http.Request, request provider.Request) {
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

	pathname := request.SubPath(name)
	info, err := s.storage.Stat(ctx, pathname)
	if err != nil {
		s.error(w, r, request, model.WrapNotFound(err))
		return
	}

	deletePath := info.Pathname
	if info.IsDir() {
		deletePath = provider.Dirname(deletePath)
	}

	if err = s.storage.RemoveAll(ctx, deletePath); err != nil {
		s.error(w, r, request, model.WrapInternal(err))
		return
	}

	if info.IsDir() {
		request = request.DeletePreference(pathname)
		provider.SetPrefsCookie(w, request)
	}

	go s.pushEvent(context.WithoutCancel(ctx), provider.NewDeleteEvent(ctx, request, info, s.renderer))

	s.renderer.Redirect(w, r, fmt.Sprintf("?d=%s", request.Display), renderer.NewSuccessMessage("%s successfully deleted", info.Name()))
}

func (s *Service) DeleteSavedSearch(w http.ResponseWriter, r *http.Request, request provider.Request) {
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

	item, err := s.storage.Stat(ctx, request.Filepath())
	if err != nil {
		s.error(w, r, request, model.WrapNotFound(err))
		return
	}

	if err = s.searchService.Delete(ctx, item, name); err != nil {
		s.error(w, r, request, err)
		return
	}

	s.renderer.Redirect(w, r, fmt.Sprintf("?d=%s", request.Display), renderer.NewSuccessMessage("%s successfully deleted", name))
}
