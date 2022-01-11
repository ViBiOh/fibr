package crud

import (
	"fmt"
	"net/http"
	"path"
	"strings"

	absto "github.com/ViBiOh/absto/pkg/model"
	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/httputils/v4/pkg/model"
	"github.com/ViBiOh/httputils/v4/pkg/renderer"
	"golang.org/x/crypto/bcrypt"
)

func (a App) bestSharePath(request provider.Request, name string) string {
	if !request.Share.IsZero() {
		return request.AbsoluteURL(name)
	}

	var remainingPath string
	var bestShare provider.Share

	for _, share := range a.shareApp.List() {
		if !strings.HasPrefix(request.Path, share.Path) {
			continue
		}

		if bestShare.IsZero() {
			bestShare = share
			remainingPath = strings.TrimPrefix(request.Path, share.Path)
			continue
		}

		if len(share.Password) > 0 && len(bestShare.Password) == 0 {
			continue
		}

		newRemainingPath := strings.TrimPrefix(request.Path, share.Path)
		if len(newRemainingPath) > len(remainingPath) {
			continue
		}

		bestShare = share
		remainingPath = newRemainingPath
	}

	if !bestShare.IsZero() {
		return provider.URL(remainingPath, name, bestShare)
	}

	return request.AbsoluteURL(name)
}

func (a App) createShare(w http.ResponseWriter, r *http.Request, request provider.Request) {
	if !request.CanShare {
		a.error(w, r, request, model.WrapForbidden(ErrNotAuthorized))
		return
	}

	var err error

	edit, err := getFormBool(r.FormValue("edit"))
	if err != nil {
		a.error(w, r, request, model.WrapInvalid(err))
		return
	}

	duration, err := getFormDuration(r.FormValue("duration"))
	if err != nil {
		a.error(w, r, request, model.WrapInvalid(err))
		return
	}

	password := ""
	if passwordValue := strings.TrimSpace(r.FormValue("password")); passwordValue != "" {
		hash, err := bcrypt.GenerateFromPassword([]byte(passwordValue), a.bcryptCost)
		if err != nil {
			a.error(w, r, request, model.WrapInternal(err))
			return
		}

		password = string(hash)
	}

	info, err := a.storageApp.Info(request.Path)
	if err != nil {
		if absto.IsNotExist(err) {
			a.error(w, r, request, model.WrapNotFound(err))
		} else {
			a.error(w, r, request, model.WrapInternal(err))
		}
		return
	}

	id, err := a.shareApp.Create(request.Path, edit, password, info.IsDir, duration)
	if err != nil {
		a.error(w, r, request, model.WrapInternal(err))
		return
	}

	if r.Header.Get("Accept") == "text/plain" {
		w.WriteHeader(http.StatusCreated)
		provider.SafeWrite(w, id)

		return
	}

	redirection := request.Path
	if !info.IsDir {
		redirection = fmt.Sprintf("%s/", path.Dir(redirection))
	}

	a.rendererApp.Redirect(w, r, fmt.Sprintf("%s/?d=%s#share-list", redirection, request.LayoutPath(redirection)), renderer.NewSuccessMessage("Share successfully created with ID: %s", id))
}

func (a App) deleteShare(w http.ResponseWriter, r *http.Request, request provider.Request) {
	if !request.CanShare {
		a.error(w, r, request, model.WrapForbidden(ErrNotAuthorized))
		return
	}

	id := r.FormValue("id")

	if err := a.shareApp.Delete(id); err != nil {
		a.error(w, r, request, model.WrapInternal(err))
		return
	}

	if r.Header.Get("Accept") == "text/plain" {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	a.rendererApp.Redirect(w, r, fmt.Sprintf("%s?d=%s#share-list", request.Path, request.Display), renderer.NewSuccessMessage("Share with id %s successfully deleted", id))
}
