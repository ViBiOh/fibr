package crud

import (
	"crypto/rand"
	"fmt"
	"net/http"
	"path"
	"strings"
	"time"

	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/fibr/pkg/sha"
	"github.com/ViBiOh/httputils/v4/pkg/renderer"
	"golang.org/x/crypto/bcrypt"
)

func uuid() (string, error) {
	raw := make([]byte, 16)
	_, _ = rand.Read(raw)

	raw[8] = raw[8]&^0xc0 | 0x80
	raw[6] = raw[6]&^0xf0 | 0x40

	return fmt.Sprintf("%x-%x-%x-%x-%x", raw[0:4], raw[4:6], raw[6:8], raw[8:10], raw[10:]), nil
}

func (a *app) generateShareID() (string, error) {
	for {
		uuid, err := uuid()
		if err != nil {
			return "", err
		}
		id := sha.Sha1(uuid)[:8]

		if _, ok := a.metadatas.Load(id); !ok {
			return id, nil
		}
	}
}

func (a *app) createShare(filepath string, edit bool, password string, isDir bool, duration time.Duration) (string, error) {
	id, err := a.generateShareID()
	if err != nil {
		return "", err
	}

	a.metadatas.Store(id, provider.Share{
		ID:       id,
		Path:     filepath,
		RootName: path.Base(filepath),
		Edit:     edit,
		Password: password,
		File:     !isDir,
		Creation: a.clock.Now(),
		Duration: duration,
	})

	return id, nil
}

// CreateShare create a share for given URL
func (a *app) CreateShare(w http.ResponseWriter, r *http.Request, request provider.Request) {
	if !request.CanShare {
		a.renderer.Error(w, request, provider.NewError(http.StatusForbidden, ErrNotAuthorized))
		return
	}

	var err error

	edit, err := getFormBool(r.FormValue("edit"))
	if err != nil {
		a.renderer.Error(w, request, provider.NewError(http.StatusBadRequest, err))
		return
	}

	duration, err := getFormDuration(r.FormValue("duration"))
	if err != nil {
		a.renderer.Error(w, request, provider.NewError(http.StatusBadRequest, err))
		return
	}

	password := ""
	if passwordValue := strings.TrimSpace(r.FormValue("password")); passwordValue != "" {
		hash, err := bcrypt.GenerateFromPassword([]byte(passwordValue), 12)
		if err != nil {
			a.renderer.Error(w, request, provider.NewError(http.StatusInternalServerError, err))
			return
		}

		password = string(hash)
	}

	info, err := a.storage.Info(request.Path)
	if err != nil {
		if provider.IsNotExist(err) {
			a.renderer.Error(w, request, provider.NewError(http.StatusNotFound, err))
		} else {
			a.renderer.Error(w, request, provider.NewError(http.StatusInternalServerError, err))
		}
		return
	}

	id, err := a.createShare(request.Path, edit, password, info.IsDir, duration)
	if err != nil {
		a.renderer.Error(w, request, provider.NewError(http.StatusInternalServerError, err))
		return
	}

	if err = a.saveMetadata(); err != nil {
		a.renderer.Error(w, request, provider.NewError(http.StatusInternalServerError, err))
		return
	}

	if r.Header.Get("Accept") == "text/plain" {
		w.WriteHeader(http.StatusCreated)
		provider.SafeWrite(w, id)

		return
	}

	http.Redirect(w, r, fmt.Sprintf("%s/?%s#share-list", path.Dir(request.GetURI("")), renderer.NewSuccessMessage(fmt.Sprintf("Share successfully created with ID: %s", id))), http.StatusFound)
}

// DeleteShare delete a share from given ID
func (a *app) DeleteShare(w http.ResponseWriter, r *http.Request, request provider.Request) {
	if !request.CanShare {
		a.renderer.Error(w, request, provider.NewError(http.StatusForbidden, ErrNotAuthorized))
		return
	}

	id := r.FormValue("id")
	a.metadatas.Delete(id)

	if err := a.saveMetadata(); err != nil {
		a.renderer.Error(w, request, provider.NewError(http.StatusInternalServerError, err))
		return
	}

	if r.Header.Get("Accept") == "text/plain" {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	http.Redirect(w, r, fmt.Sprintf("%s/?%s#share-list", request.GetURI(""), renderer.NewSuccessMessage(fmt.Sprintf("Share with id %s successfully deleted", id))), http.StatusFound)
}
