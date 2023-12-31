package crud

import (
	"errors"
	"fmt"
	"net/http"
	"path"
	"strings"

	absto "github.com/ViBiOh/absto/pkg/model"
	"github.com/ViBiOh/auth/v2/pkg/argon"
	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/httputils/v4/pkg/model"
	"github.com/ViBiOh/httputils/v4/pkg/renderer"
)

func (s Service) bestSharePath(pathname string) string {
	var remainingPath string
	var bestShare provider.Share

	for _, share := range s.share.List() {
		if !strings.HasPrefix(pathname, share.Path) {
			continue
		}

		if bestShare.IsZero() {
			bestShare = share
			remainingPath = strings.TrimPrefix(pathname, share.Path)
			continue
		}

		if len(bestShare.Password) == 0 && len(share.Password) != 0 {
			continue
		}

		newRemainingPath := strings.TrimPrefix(pathname, share.Path)
		if len(newRemainingPath) > len(remainingPath) {
			continue
		}

		bestShare = share
		remainingPath = newRemainingPath
	}

	if !bestShare.IsZero() {
		return provider.URL(remainingPath, "", bestShare)
	}

	return ""
}

func parseRights(value string) (edit, story bool, err error) {
	switch value {
	case "edit":
		return true, false, nil
	case "read":
		return false, false, nil
	case "story":
		return false, true, nil
	default:
		return false, false, errors.New("invalid rights: edit, read or story allowed")
	}
}

func (s Service) createShare(w http.ResponseWriter, r *http.Request, request provider.Request) {
	var err error

	edit, story, err := parseRights(strings.TrimSpace(r.FormValue("rights")))
	if err != nil {
		s.error(w, r, request, model.WrapInvalid(err))
		return
	}

	duration, err := getFormDuration(r.FormValue("duration"))
	if err != nil {
		s.error(w, r, request, model.WrapInvalid(err))
		return
	}

	password := ""
	if passwordValue := strings.TrimSpace(r.FormValue("password")); passwordValue != "" {
		hash, err := argon.GenerateFromPassword(passwordValue)
		if err != nil {
			s.error(w, r, request, model.WrapInternal(err))
			return
		}

		password = hash
	}

	ctx := r.Context()

	info, err := s.storage.Stat(ctx, request.Filepath())
	if err != nil {
		if absto.IsNotExist(err) {
			s.error(w, r, request, model.WrapNotFound(err))
		} else {
			s.error(w, r, request, model.WrapInternal(err))
		}
		return
	}

	id, err := s.share.Create(ctx, request.Filepath(), edit, story, password, info.IsDir(), duration)
	if err != nil {
		s.error(w, r, request, model.WrapInternal(err))
		return
	}

	if r.Header.Get("Accept") == "text/plain" {
		w.WriteHeader(http.StatusCreated)
		provider.SafeWrite(ctx, w, id)

		return
	}

	redirection := request.Filepath()
	if !info.IsDir() {
		redirection = fmt.Sprintf("%s/", path.Dir(redirection))
	}

	s.renderer.Redirect(w, r, fmt.Sprintf("%s/?d=%s#share-list", redirection, request.LayoutPath(redirection)), renderer.NewSuccessMessage("Share successfully created with ID: %s", id))
}

func (s Service) deleteShare(w http.ResponseWriter, r *http.Request, request provider.Request) {
	id := r.FormValue("id")

	if err := s.share.Delete(r.Context(), id); err != nil {
		s.error(w, r, request, model.WrapInternal(err))
		return
	}

	if r.Header.Get("Accept") == "text/plain" {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	s.renderer.Redirect(w, r, fmt.Sprintf("%s?d=%s#share-list", request.Path, request.Display), renderer.NewSuccessMessage("Share with id %s successfully deleted", id))
}
