package crud

import (
	"errors"
	"fmt"
	"net/http"
	"path"

	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/httputils/v4/pkg/renderer"
)

func (a *app) doRename(oldPath, newPath string, oldItem provider.StorageItem) (provider.StorageItem, error) {
	if err := a.storageApp.Rename(oldPath, newPath); err != nil {
		return provider.StorageItem{}, err
	}

	newItem, err := a.storageApp.Info(newPath)
	if err != nil {
		return provider.StorageItem{}, err
	}

	if err := a.metadataApp.RenameSharePath(oldPath, newPath); err != nil {
		return newItem, fmt.Errorf("error while updating metadatas: %s", err)
	}

	go a.thumbnailApp.Rename(oldItem, newItem)

	return newItem, nil
}

// Rename rename given path to a new one
func (a *app) Rename(w http.ResponseWriter, r *http.Request, request provider.Request) {
	if !request.CanEdit {
		a.rendererApp.Error(w, request, provider.NewError(http.StatusForbidden, ErrNotAuthorized))
		return
	}

	oldName, httErr := checkFormName(r, "name")
	if httErr != nil && httErr.Err != ErrEmptyName {
		a.rendererApp.Error(w, request, httErr)
		return
	}

	newFolder, err := getNewFolder(r, request)
	if err != nil {
		a.rendererApp.Error(w, request, provider.NewError(http.StatusInternalServerError, err))
		return
	}

	newName, err := getNewName(r)
	if err != nil {
		a.rendererApp.Error(w, request, provider.NewError(http.StatusInternalServerError, err))
		return
	}

	oldPath := request.GetFilepath(oldName)
	newPath := provider.GetPathname(newFolder, newName, request.Share)

	if _, err := a.storageApp.Info(newPath); err == nil {
		a.rendererApp.Error(w, request, provider.NewError(http.StatusBadRequest, errors.New("new name already exist")))
		return
	} else if !provider.IsNotExist(err) {
		a.rendererApp.Error(w, request, provider.NewError(http.StatusInternalServerError, err))
		return
	}

	oldItem, err := a.storageApp.Info(oldPath)
	if err != nil {
		if !provider.IsNotExist(err) {
			a.rendererApp.Error(w, request, provider.NewError(http.StatusInternalServerError, err))
		} else {
			a.rendererApp.Error(w, request, provider.NewError(http.StatusNotFound, err))
		}

		return
	}

	newItem, err := a.doRename(oldPath, newPath, oldItem)
	if err != nil {
		a.rendererApp.Error(w, request, provider.NewError(http.StatusInternalServerError, err))
		return
	}

	var message string
	uri := request.GetURI("")

	if newFolder != uri {
		message = fmt.Sprintf("%s successfully moved to %s", oldItem.Name, provider.GetURI(newFolder, newName, request.Share))
	} else {
		message = fmt.Sprintf("%s successfully renamed to %s", oldItem.Name, newItem.Name)
	}

	http.Redirect(w, r, fmt.Sprintf("%s%s", a.publicURL, path.Join(uri, fmt.Sprintf("?%s", renderer.NewSuccessMessage(message)))), http.StatusFound)
}

func getNewFolder(r *http.Request, request provider.Request) (string, error) {
	newFolder, httpErr := checkFolderName(r.FormValue("newFolder"), request)
	if httpErr != nil {
		return "", httpErr
	}

	return provider.SanitizeName(newFolder, false)
}

func getNewName(r *http.Request) (string, error) {
	newName, httpErr := checkFormName(r, "newName")
	if httpErr != nil {
		return "", httpErr
	}

	return provider.SanitizeName(newName, true)
}
