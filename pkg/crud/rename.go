package crud

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/httputils/v4/pkg/model"
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
		a.rendererApp.Error(w, model.WrapForbidden(ErrNotAuthorized))
		return
	}

	oldName, err := checkFormName(r, "name")
	if err != nil && !errors.Is(err, ErrEmptyName) {
		a.rendererApp.Error(w, err)
		return
	}

	newFolder, err := getNewFolder(r, request)
	if err != nil {
		a.rendererApp.Error(w, model.WrapInternal(err))
		return
	}

	newName, err := getNewName(r)
	if err != nil {
		a.rendererApp.Error(w, model.WrapInternal(err))
		return
	}

	oldPath := request.GetFilepath(oldName)
	newPath := provider.GetPathname(newFolder, newName, request.Share)

	if _, err := a.storageApp.Info(newPath); err == nil {
		a.rendererApp.Error(w, model.WrapInvalid(errors.New("new name already exist")))
		return
	} else if !provider.IsNotExist(err) {
		a.rendererApp.Error(w, model.WrapInternal(err))
		return
	}

	oldItem, err := a.storageApp.Info(oldPath)
	if err != nil {
		if !provider.IsNotExist(err) {
			err = model.WrapInternal(err)
		} else {
			err = model.WrapNotFound(err)
		}

		a.rendererApp.Error(w, err)
		return
	}

	newItem, err := a.doRename(oldPath, newPath, oldItem)
	if err != nil {
		a.rendererApp.Error(w, model.WrapInternal(err))
		return
	}

	if oldItem.IsDir {
		updatePreferences(request, oldPath, newPath)
		provider.SetPrefsCookie(w, request)
	}

	var message string
	uri := request.URL("")

	if newFolder != uri {
		message = fmt.Sprintf("%s successfully moved to %s", oldItem.Name, provider.URL(newFolder, newName, request.Share))
	} else {
		message = fmt.Sprintf("%s successfully renamed to %s", oldItem.Name, newItem.Name)
	}

	a.rendererApp.Redirect(w, r, fmt.Sprintf("%s/?d=%s", uri, request.Layout("")), renderer.NewSuccessMessage(message))
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

func updatePreferences(request provider.Request, oldPath, newPath string) {
	paths := request.Preferences.ListLayoutPath

	for index, layoutPath := range paths {
		if layoutPath != oldPath {
			paths[index] = newPath
		} else {
			paths[index] = layoutPath
		}
	}
}
