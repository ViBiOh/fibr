package crud

import (
	"fmt"
	"net/http"

	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/httputils/pkg/logger"
)

// Delete given path from filesystem
func (a *app) Delete(w http.ResponseWriter, r *http.Request, request *provider.Request) {
	if !request.CanEdit {
		a.renderer.Error(w, provider.NewError(http.StatusForbidden, ErrNotAuthorized))
		return
	}

	name, httpErr := checkFormName(r, "name")
	if httpErr != nil && httpErr.Err != ErrEmptyName {
		a.renderer.Error(w, httpErr)
		return
	}

	info, err := a.storage.Info(request.GetFilepath(name))
	if err != nil {
		a.renderer.Error(w, provider.NewError(http.StatusNotFound, err))
		return
	}

	if err := a.storage.Remove(info.Pathname); err != nil {
		a.renderer.Error(w, provider.NewError(http.StatusInternalServerError, err))
		return
	}

	go a.deleteThumbnail(info.Pathname)

	a.List(w, request, r.URL.Query().Get("d"), &provider.Message{Level: "success", Content: fmt.Sprintf("%s successfully deleted", info.Name)})
}

func (a *app) deleteThumbnail(path string) bool {
	thumbnailPath, ok := a.thumbnail.HasThumbnail(path)
	if !ok {
		return false
	}

	if err := a.storage.Remove(thumbnailPath); err != nil {
		logger.Error("%#v", err)
	}

	return true
}
