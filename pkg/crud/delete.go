package crud

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/httputils/v4/pkg/renderer"
)

// Delete given path from filesystem
func (a *app) Delete(w http.ResponseWriter, r *http.Request, request provider.Request) {
	if !request.CanEdit {
		a.renderer.Error(w, request, provider.NewError(http.StatusForbidden, ErrNotAuthorized))
		return
	}

	name, httpErr := checkFormName(r, "name")
	if httpErr != nil && httpErr.Err != ErrEmptyName {
		a.renderer.Error(w, request, httpErr)
		return
	}

	info, err := a.storage.Info(request.GetFilepath(name))
	if err != nil {
		a.renderer.Error(w, request, provider.NewError(http.StatusNotFound, err))
		return
	}

	if err := a.storage.Remove(info.Pathname); err != nil {
		a.renderer.Error(w, request, provider.NewError(http.StatusInternalServerError, err))
		return
	}

	a.metadatas.Range(func(key interface{}, value interface{}) bool {
		share := value.(provider.Share)

		if strings.HasPrefix(share.Path, info.Pathname) {
			a.metadatas.Delete(key)
		}

		return true
	})

	if err := a.saveMetadata(); err != nil {
		a.renderer.Error(w, request, provider.NewError(http.StatusInternalServerError, err))
		return
	}

	go a.thumbnail.Remove(info)

	http.Redirect(w, r, fmt.Sprintf("%s/?%s", request.GetURI(""), renderer.NewSuccessMessage(fmt.Sprintf("%s successfully deleted", info.Name))), http.StatusFound)
}
