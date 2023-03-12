package crud

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/httputils/v4/pkg/cntxt"
	"github.com/ViBiOh/httputils/v4/pkg/model"
	"github.com/ViBiOh/httputils/v4/pkg/renderer"
)

func (a App) Delete(w http.ResponseWriter, r *http.Request, request provider.Request) {
	if !request.CanEdit {
		a.error(w, r, request, model.WrapForbidden(ErrNotAuthorized))
		return
	}

	name, err := checkFormName(r, "name")
	if err != nil && !errors.Is(err, ErrEmptyName) {
		a.error(w, r, request, err)
		return
	}

	ctx := r.Context()

	pathname := request.SubPath(name)
	info, err := a.storageApp.Info(ctx, pathname)
	if err != nil {
		a.error(w, r, request, model.WrapNotFound(err))
		return
	}

	deletePath := info.Pathname
	if info.IsDir {
		deletePath = provider.Dirname(deletePath)
	}

	if err = a.storageApp.Remove(ctx, deletePath); err != nil {
		a.error(w, r, request, model.WrapInternal(err))
		return
	}

	if info.IsDir {
		request = request.DeletePreference(pathname)
		provider.SetPrefsCookie(w, request)
	}

	go a.notify(cntxt.WithoutDeadline(ctx), provider.NewDeleteEvent(request, info, a.rendererApp))

	a.rendererApp.Redirect(w, r, fmt.Sprintf("?d=%s", request.Display), renderer.NewSuccessMessage("%s successfully deleted", info.Name))
}

func (a App) DeleteSavedSearch(w http.ResponseWriter, r *http.Request, request provider.Request) {
	if !request.CanEdit {
		a.error(w, r, request, model.WrapForbidden(ErrNotAuthorized))
		return
	}

	name, err := checkFormName(r, "name")
	if err != nil && !errors.Is(err, ErrEmptyName) {
		a.error(w, r, request, err)
		return
	}

	ctx := r.Context()

	item, err := a.storageApp.Info(ctx, request.Filepath())
	if err != nil {
		a.error(w, r, request, model.WrapNotFound(err))
		return
	}

	if err = a.searchApp.Delete(ctx, item, name); err != nil {
		a.error(w, r, request, err)
		return
	}

	a.rendererApp.Redirect(w, r, fmt.Sprintf("?d=%s", request.Display), renderer.NewSuccessMessage("%s successfully deleted", name))
}
