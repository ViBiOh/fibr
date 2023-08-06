package crud

import (
	"errors"
	"fmt"
	"net/http"
	"path"

	absto "github.com/ViBiOh/absto/pkg/model"
	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/httputils/v4/pkg/model"
	"github.com/ViBiOh/httputils/v4/pkg/renderer"
)

func (a App) Create(w http.ResponseWriter, r *http.Request, request provider.Request) {
	if !request.CanEdit {
		a.error(w, r, request, model.WrapForbidden(ErrNotAuthorized))
		return
	}

	name, err := checkFormName(r, "name")
	if err != nil && !errors.Is(err, ErrEmptyName) {
		a.error(w, r, request, err)
		return
	}

	name, err = provider.SanitizeName(name, false)
	if err != nil {
		a.error(w, r, request, model.WrapInternal(err))
		return
	}

	pathname := request.SubPath(name)

	if err = a.storageApp.Mkdir(r.Context(), pathname, absto.DirectoryPerm); err != nil {
		a.error(w, r, request, model.WrapInternal(err))
		return
	}

	a.rendererApp.Redirect(w, r, fmt.Sprintf("%s/?d=%s", name, request.Display), renderer.NewSuccessMessage("Directory %s successfully created", path.Base(pathname)))
}

func (a App) CreateSavedSearch(w http.ResponseWriter, r *http.Request, request provider.Request) {
	if !request.CanEdit {
		a.error(w, r, request, model.WrapForbidden(ErrNotAuthorized))
		return
	}

	name, err := checkFormName(r, "name")
	if err != nil && !errors.Is(err, ErrEmptyName) {
		a.error(w, r, request, err)
		return
	}

	name, err = provider.SanitizeName(name, false)
	if err != nil {
		a.error(w, r, request, model.WrapInternal(err))
		return
	}

	ctx := r.Context()

	item, err := a.storageApp.Stat(ctx, request.Filepath())
	if err != nil {
		a.error(w, r, request, model.WrapNotFound(err))
		return
	}

	if err = a.searchApp.Add(ctx, item, provider.Search{
		ID:    provider.Hash(name),
		Name:  name,
		Query: r.URL.RawQuery,
	}); err != nil {
		a.error(w, r, request, fmt.Errorf("update: %w", err))
		return
	}

	a.rendererApp.Redirect(w, r, fmt.Sprintf("?%s", r.URL.RawQuery), renderer.NewSuccessMessage("Saved search %s successfully created", name))
}
