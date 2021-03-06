package crud

import (
	"errors"
	"fmt"
	"net/http"
	"path"
	"strings"

	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/httputils/v4/pkg/model"
	"github.com/ViBiOh/httputils/v4/pkg/renderer"
	"golang.org/x/crypto/bcrypt"
)

// CreateShare create a share for given URL
func (a *app) CreateShare(w http.ResponseWriter, r *http.Request, request provider.Request) {
	if !a.metadataApp.Enabled() {
		a.rendererApp.Error(w, model.WrapInternal(errors.New("metadatas are disabled")))
		return
	}

	if !request.CanShare {
		a.rendererApp.Error(w, model.WrapForbidden(ErrNotAuthorized))
		return
	}

	var err error

	edit, err := getFormBool(r.FormValue("edit"))
	if err != nil {
		a.rendererApp.Error(w, model.WrapInvalid(err))
		return
	}

	duration, err := getFormDuration(r.FormValue("duration"))
	if err != nil {
		a.rendererApp.Error(w, model.WrapInvalid(err))
		return
	}

	password := ""
	if passwordValue := strings.TrimSpace(r.FormValue("password")); passwordValue != "" {
		hash, err := bcrypt.GenerateFromPassword([]byte(passwordValue), provider.BcryptCost)
		if err != nil {
			a.rendererApp.Error(w, model.WrapInternal(err))
			return
		}

		password = string(hash)
	}

	info, err := a.storageApp.Info(request.Path)
	if err != nil {
		if provider.IsNotExist(err) {
			a.rendererApp.Error(w, model.WrapNotFound(err))
		} else {
			a.rendererApp.Error(w, model.WrapInternal(err))
		}
		return
	}

	id, err := a.metadataApp.CreateShare(request.Path, edit, password, info.IsDir, duration)
	if err != nil {
		a.rendererApp.Error(w, model.WrapInternal(err))
		return
	}

	if r.Header.Get("Accept") == "text/plain" {
		w.WriteHeader(http.StatusCreated)
		provider.SafeWrite(w, id)

		return
	}

	a.rendererApp.Redirect(w, r, fmt.Sprintf("%s/?d=%s#share-list", path.Dir(request.URL("")), request.Layout("")), renderer.NewSuccessMessage("Share successfully created with ID: %s", id))
}

// DeleteShare delete a share from given ID
func (a *app) DeleteShare(w http.ResponseWriter, r *http.Request, request provider.Request) {
	if !a.metadataApp.Enabled() {
		a.rendererApp.Error(w, model.WrapInternal(errors.New("metadatas are disabled")))
		return
	}

	if !request.CanShare {
		a.rendererApp.Error(w, model.WrapForbidden(ErrNotAuthorized))
		return
	}

	id := r.FormValue("id")

	if err := a.metadataApp.DeleteShare(id); err != nil {
		a.rendererApp.Error(w, model.WrapInternal(err))
		return
	}

	if r.Header.Get("Accept") == "text/plain" {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	a.rendererApp.Redirect(w, r, fmt.Sprintf("%s/?d=%s#share-list", request.URL(""), request.Layout("")), renderer.NewSuccessMessage("Share with id %s successfully deleted", id))
}
