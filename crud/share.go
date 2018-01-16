package crud

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/ViBiOh/fibr/provider"
	"github.com/ViBiOh/httputils/uuid"
)

func (a *App) getShareOfPath(pathname string) []*Share {
	shares := make([]*Share, 0)

	for _, share := range a.metadatas {
		if share.Path == pathname {
			shares = append(shares, share)
		}
	}

	return shares
}

// CreateShare create a share for given URL
func (a *App) CreateShare(w http.ResponseWriter, r *http.Request, config *provider.RequestConfig) {
	if !config.CanShare {
		a.renderer.Error(w, http.StatusForbidden, errors.New(`You're not authorized to do this ⛔`))
	}

	var edit bool
	var err error

	if r.FormValue(`edit`) != `` {
		edit, err = strconv.ParseBool(r.FormValue(`edit`))
		if err != nil {
			a.renderer.Error(w, http.StatusBadRequest, fmt.Errorf(`Error while reading form: %v`, err))
			return
		}
	} else {
		edit = false
	}

	uuid, err := uuid.New()
	if err != nil {
		a.renderer.Error(w, http.StatusInternalServerError, fmt.Errorf(`Error while generating UUID: %v`, err))
		return
	}

	a.metadataLock.Lock()
	defer a.metadataLock.Unlock()

	a.metadatas = append(a.metadatas, &Share{
		ID:   uuid,
		Path: r.URL.Path,
		Edit: edit,
	})

	if err = a.saveMetadata(); err != nil {
		a.renderer.Error(w, http.StatusInternalServerError, fmt.Errorf(`Error while saving share: %v`, err))
		return
	}

	a.Get(w, r, config, &provider.Message{
		Level:   `success`,
		Content: fmt.Sprintf(`Share successfully created with ID: %s`, uuid),
	})
}
