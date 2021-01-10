package crud

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/httputils/v3/pkg/renderer"
)

func (a *app) doRename(oldPath, newPath string, oldItem provider.StorageItem) (provider.StorageItem, error) {
	if err := a.storage.Rename(oldPath, newPath); err != nil {
		return provider.StorageItem{}, err
	}

	newItem, err := a.storage.Info(newPath)
	if err != nil {
		return provider.StorageItem{}, err
	}

	a.metadataLock.Lock()
	defer a.metadataLock.Unlock()

	for _, metadata := range a.metadatas {
		if strings.HasPrefix(metadata.Path, oldPath) {
			metadata.Path = strings.Replace(metadata.Path, oldPath, newPath, 1)
		}
	}

	if err := a.saveMetadata(); err != nil {
		return newItem, fmt.Errorf("error while updating metadatas: %s", err)
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

	newFolder, httpErr := checkFolderName(r.FormValue("newFolder"), request)
	if httpErr != nil {
		a.renderer.Error(w, request, httpErr)
		return
	}

	newName, httpErr := checkFormName(r, "newName")
	if httpErr != nil {
		a.renderer.Error(w, request, httpErr)
		return
	}

	newSafeFolder, err := provider.SanitizeName(newFolder, false)
	if err != nil {
		a.renderer.Error(w, request, provider.NewError(http.StatusInternalServerError, err))
		return
	}

	newSafeName, err := provider.SanitizeName(newName, true)
	if err != nil {
		a.renderer.Error(w, request, provider.NewError(http.StatusInternalServerError, err))
		return
	}

	oldPath := request.GetFilepath(oldName)
	newPath := provider.GetPathname(newSafeFolder, newSafeName, request.Share)

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

	var message string
	uri := request.GetURI("")

	if newSafeFolder != uri {
		message = fmt.Sprintf("%s successfully moved to %s", oldItem.Name, provider.GetURI(newSafeFolder, newSafeName, request.Share))
	} else {
		message = fmt.Sprintf("%s successfully renamed to %s", oldItem.Name, newItem.Name)
	}

	http.Redirect(w, r, fmt.Sprintf("%s/?%s", uri, renderer.NewSuccessMessage(message)), http.StatusFound)
}
