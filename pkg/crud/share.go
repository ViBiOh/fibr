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

func (a App) bestSharePath(request provider.Request, name string) string {
	if !request.Share.IsZero() {
		return request.URL(name)
	}

	var remaingPath string
	var bestShare provider.Share

	for _, share := range a.shareApp.List() {
		if !strings.HasPrefix(request.Path, share.Path) {
			continue
		}

		if bestShare.IsZero() {
			bestShare = share
			remaingPath = strings.TrimPrefix(request.Path, share.Path)
			continue
		}

		if len(share.Password) > 0 && len(bestShare.Password) == 0 {
			continue
		}

		newRemainingPath := strings.TrimPrefix(request.Path, share.Path)
		if len(newRemainingPath) > len(remaingPath) {
			continue
		}

		bestShare = share
		remaingPath = newRemainingPath
	}

	if !bestShare.IsZero() {
		return provider.URL(remaingPath, name, bestShare)
	}

	return request.URL(name)
}

func (a App) createShare(w http.ResponseWriter, r *http.Request, request provider.Request) {
	if !a.shareApp.Enabled() {
		a.rendererApp.Error(w, r, model.WrapInternal(errors.New("share is disabled")))
		return
	}

	if !request.CanShare {
		a.rendererApp.Error(w, r, model.WrapForbidden(ErrNotAuthorized))
		return
	}

	var err error

	edit, err := getFormBool(r.FormValue("edit"))
	if err != nil {
		a.rendererApp.Error(w, r, model.WrapInvalid(err))
		return
	}

	duration, err := getFormDuration(r.FormValue("duration"))
	if err != nil {
		a.rendererApp.Error(w, r, model.WrapInvalid(err))
		return
	}

	password := ""
	if passwordValue := strings.TrimSpace(r.FormValue("password")); passwordValue != "" {
		hash, err := bcrypt.GenerateFromPassword([]byte(passwordValue), a.bcryptCost)
		if err != nil {
			a.rendererApp.Error(w, r, model.WrapInternal(err))
			return
		}

		password = string(hash)
	}

	info, err := a.storageApp.Info(request.Path)
	if err != nil {
		if provider.IsNotExist(err) {
			a.rendererApp.Error(w, r, model.WrapNotFound(err))
		} else {
			a.rendererApp.Error(w, r, model.WrapInternal(err))
		}
		return
	}

	id, err := a.shareApp.Create(request.Path, edit, password, info.IsDir, duration)
	if err != nil {
		a.rendererApp.Error(w, r, model.WrapInternal(err))
		return
	}

	if r.Header.Get("Accept") == "text/plain" {
		w.WriteHeader(http.StatusCreated)
		provider.SafeWrite(w, id)

		return
	}

	redirection := request.URL("")
	if !info.IsDir {
		redirection = path.Dir(redirection)
	}

	a.rendererApp.Redirect(w, r, fmt.Sprintf("%s/?d=%s#share-list", redirection, request.LayoutPath(redirection)), renderer.NewSuccessMessage("Share successfully created with ID: %s", id))
}

func (a App) deleteShare(w http.ResponseWriter, r *http.Request, request provider.Request) {
	if !a.shareApp.Enabled() {
		a.rendererApp.Error(w, r, model.WrapInternal(errors.New("share is disabled")))
		return
	}

	if !request.CanShare {
		a.rendererApp.Error(w, r, model.WrapForbidden(ErrNotAuthorized))
		return
	}

	id := r.FormValue("id")

	if err := a.shareApp.Delete(id); err != nil {
		a.rendererApp.Error(w, r, model.WrapInternal(err))
		return
	}

	if r.Header.Get("Accept") == "text/plain" {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	a.rendererApp.Redirect(w, r, fmt.Sprintf("%s/?d=%s#share-list", request.URL(""), request.Layout("")), renderer.NewSuccessMessage("Share with id %s successfully deleted", id))
}
