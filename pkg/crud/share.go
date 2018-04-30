package crud

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/httputils/pkg/tools"
	"github.com/ViBiOh/httputils/pkg/uuid"
	"golang.org/x/crypto/bcrypt"
)

// CreateShare create a share for given URL
func (a *App) CreateShare(w http.ResponseWriter, r *http.Request, config *provider.RequestConfig) {
	if !config.CanShare {
		a.renderer.Error(w, http.StatusForbidden, ErrNotAuthorized)
	}

	var err error

	edit := false
	if editValue := r.FormValue(`edit`); editValue != `` {
		edit, err = strconv.ParseBool(editValue)
		if err != nil {
			a.renderer.Error(w, http.StatusBadRequest, fmt.Errorf(`Error while reading form: %v`, err))
			return
		}
	}

	password := ``
	if passwordValue := r.FormValue(`password`); passwordValue != `` {
		hash, err := bcrypt.GenerateFromPassword([]byte(passwordValue), 12)
		if err != nil {
			a.renderer.Error(w, http.StatusInternalServerError, fmt.Errorf(`Error while hashing password: %v`, err))
			return
		}

		password = string(hash)
	}

	uuid, err := uuid.New()
	if err != nil {
		a.renderer.Error(w, http.StatusInternalServerError, fmt.Errorf(`Error while generating UUID: %v`, err))
		return
	}
	id := tools.Sha1(uuid)[:8]

	a.metadataLock.Lock()
	defer a.metadataLock.Unlock()

	a.metadatas = append(a.metadatas, &Share{
		ID:       id,
		Path:     r.URL.Path,
		Edit:     edit,
		Password: password,
	})

	if err = a.saveMetadata(); err != nil {
		a.renderer.Error(w, http.StatusInternalServerError, fmt.Errorf(`Error while saving share: %v`, err))
		return
	}

	a.GetWithMessage(w, r, config, &provider.Message{
		Level:   `success`,
		Content: fmt.Sprintf(`Share successfully created with ID: %s`, id),
	})
}

// DeleteShare delete a share from given ID
func (a *App) DeleteShare(w http.ResponseWriter, r *http.Request, config *provider.RequestConfig) {
	if !config.CanShare {
		a.renderer.Error(w, http.StatusForbidden, ErrNotAuthorized)
	}

	id := r.FormValue(`id`)

	a.metadataLock.Lock()
	defer a.metadataLock.Unlock()

	for i, metadata := range a.metadatas {
		if metadata.ID == id {
			a.metadatas = append(a.metadatas[:i], a.metadatas[i+1:]...)
			break
		}
	}

	if err := a.saveMetadata(); err != nil {
		a.renderer.Error(w, http.StatusInternalServerError, fmt.Errorf(`Error while saving share: %v`, err))
		return
	}

	a.GetWithMessage(w, r, config, &provider.Message{
		Level:   `success`,
		Content: fmt.Sprintf(`Share with id %s successfully deleted`, id),
	})
}
