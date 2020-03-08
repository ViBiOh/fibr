package crud

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"

	"github.com/ViBiOh/fibr/pkg/provider"
)

func (a *app) doRename(oldPath, newPath string, oldItem provider.StorageItem) (provider.StorageItem, error) {
	if err := a.storage.Rename(oldPath, newPath); err != nil {
		return provider.StorageItem{}, err
	}

	newItem, err := a.storage.Info(newPath)
	if err != nil {
		return provider.StorageItem{}, err
	}

	go a.thumbnail.Rename(oldItem, newItem)

	return newItem, nil
}

// Rename rename given path to a new one
func (a *app) Rename(w http.ResponseWriter, r *http.Request, request provider.Request) {
	if !request.CanEdit {
		a.renderer.Error(w, request, provider.NewError(http.StatusForbidden, ErrNotAuthorized))
		return
	}

	oldName, httErr := checkFormName(r, "name")
	if httErr != nil && httErr.Err != ErrEmptyName {
		a.renderer.Error(w, request, httErr)
		return
	}

	newName, httpErr := checkFormName(r, "newName")
	if httpErr != nil {
		a.renderer.Error(w, request, httpErr)
		return
	}

	newSafeName, err := provider.SanitizeName(newName, true)
	if err != nil {
		a.renderer.Error(w, request, provider.NewError(http.StatusInternalServerError, err))
		return
	}

	oldPath := request.GetFilepath(oldName)
	newPath := request.GetFilepath(newSafeName)

	if _, err := a.storage.Info(newPath); err == nil {
		a.renderer.Error(w, request, provider.NewError(http.StatusBadRequest, errors.New("new name already exist")))
		return
	} else if !provider.IsNotExist(err) {
		a.renderer.Error(w, request, provider.NewError(http.StatusInternalServerError, err))
		return
	}

	oldItem, err := a.storage.Info(oldPath)
	if err != nil {
		if !provider.IsNotExist(err) {
			a.renderer.Error(w, request, provider.NewError(http.StatusInternalServerError, err))
		} else {
			a.renderer.Error(w, request, provider.NewError(http.StatusNotFound, err))
		}

		return
	}

	newItem, err := a.doRename(oldPath, newPath, oldItem)
	if err != nil {
		a.renderer.Error(w, request, provider.NewError(http.StatusInternalServerError, err))
		return
	}

	http.Redirect(w, r, fmt.Sprintf("%s/?message=%s&messageLevel=success", request.GetURI(""), url.QueryEscape(fmt.Sprintf("%s successfully renamed to %s", oldItem.Name, newItem.Name))), http.StatusFound)
}
