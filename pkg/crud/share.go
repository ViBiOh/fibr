package crud

import (
	"fmt"
	"net/http"
	"path"
	"strconv"
	"strings"

	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/fibr/pkg/sha"
	"github.com/ViBiOh/httputils/v3/pkg/uuid"
	"golang.org/x/crypto/bcrypt"
)

// CreateShare create a share for given URL
func (a *app) CreateShare(w http.ResponseWriter, r *http.Request, request *provider.Request) {
	if !request.CanShare {
		a.renderer.Error(w, provider.NewError(http.StatusForbidden, ErrNotAuthorized))
		return
	}

	var err error

	edit := false
	if editValue := strings.TrimSpace(r.FormValue("edit")); editValue != "" {
		edit, err = strconv.ParseBool(editValue)
		if err != nil {
			a.renderer.Error(w, provider.NewError(http.StatusBadRequest, err))
			return
		}
	}

	password := ""
	if passwordValue := strings.TrimSpace(r.FormValue("password")); passwordValue != "" {
		hash, err := bcrypt.GenerateFromPassword([]byte(passwordValue), 12)
		if err != nil {
			a.renderer.Error(w, provider.NewError(http.StatusInternalServerError, err))
			return
		}

		password = string(hash)
	}

	uuid, err := uuid.New()
	if err != nil {
		a.renderer.Error(w, provider.NewError(http.StatusInternalServerError, err))
		return
	}
	id := sha.Sha1(uuid)[:8]

	a.metadataLock.Lock()
	defer a.metadataLock.Unlock()

	a.metadatas = append(a.metadatas, &provider.Share{
		ID:       id,
		Path:     request.Path,
		RootName: path.Base(request.Path),
		Edit:     edit,
		Password: password,
	})

	if err = a.saveMetadata(); err != nil {
		a.renderer.Error(w, provider.NewError(http.StatusInternalServerError, err))
		return
	}

	a.GetWithMessage(w, r, request, &provider.Message{
		Level:   "success",
		Content: fmt.Sprintf("Share successfully created with ID: %s", id),
	})
}

// DeleteShare delete a share from given ID
func (a *app) DeleteShare(w http.ResponseWriter, r *http.Request, request *provider.Request) {
	if !request.CanShare {
		a.renderer.Error(w, provider.NewError(http.StatusForbidden, ErrNotAuthorized))
		return
	}

	id := r.FormValue("id")

	a.metadataLock.Lock()
	defer a.metadataLock.Unlock()

	for i, metadata := range a.metadatas {
		if metadata.ID == id {
			a.metadatas = append(a.metadatas[:i], a.metadatas[i+1:]...)
			break
		}
	}

	if err := a.saveMetadata(); err != nil {
		a.renderer.Error(w, provider.NewError(http.StatusInternalServerError, err))
		return
	}

	a.GetWithMessage(w, r, request, &provider.Message{
		Level:   "success",
		Content: fmt.Sprintf("Share with id %s successfully deleted", id),
	})
}
