package crud

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	absto "github.com/ViBiOh/absto/pkg/model"
	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/httputils/v4/pkg/model"
	"github.com/ViBiOh/httputils/v4/pkg/renderer"
)

func (a App) doRename(oldPath, newPath string, oldItem absto.Item) (absto.Item, error) {
	if err := a.storageApp.Rename(oldPath, newPath); err != nil {
		return absto.Item{}, err
	}

	newItem, err := a.storageApp.Info(newPath)
	if err != nil {
		return absto.Item{}, err
	}

	go a.notify(provider.NewRenameEvent(oldItem, newItem))

	return newItem, nil
}

// Rename rename given path to a new one
func (a App) Rename(w http.ResponseWriter, r *http.Request, request provider.Request) {
	if !request.CanEdit {
		a.error(w, r, request, model.WrapForbidden(ErrNotAuthorized))
		return
	}

	oldName, err := checkFormName(r, "name")
	if err != nil && !errors.Is(err, ErrEmptyName) {
		a.error(w, r, request, err)
		return
	}

	newFolder, err := getNewFolder(r, request)
	if err != nil {
		a.error(w, r, request, err)
		return
	}

	newName, err := getNewName(r)
	if err != nil {
		a.error(w, r, request, err)
		return
	}

	if strings.HasSuffix(oldName, "/") {
		newName = provider.Dirname(newName)
	}

	oldPath := request.SubPath(oldName)
	newPath := provider.GetPathname(newFolder, newName, request.Share)

	if _, err = a.storageApp.Info(newPath); err == nil {
		a.error(w, r, request, model.WrapInvalid(errors.New("new name already exist")))
		return
	} else if !provider.IsNotExist(err) {
		a.error(w, r, request, model.WrapInternal(err))
		return
	}

	oldItem, err := a.storageApp.Info(oldPath)
	if err != nil {
		if !provider.IsNotExist(err) {
			err = model.WrapInternal(err)
		} else {
			err = model.WrapNotFound(err)
		}

		a.error(w, r, request, err)
		return
	}

	newItem, err := a.doRename(oldPath, newPath, oldItem)
	if err != nil {
		a.error(w, r, request, model.WrapInternal(err))
		return
	}

	if oldItem.IsDir {
		updatePreferences(request, oldPath, newPath)
		provider.SetPrefsCookie(w, request)
	}

	var message string

	if newFolder != request.Path {
		message = fmt.Sprintf("%s successfully moved to %s", oldItem.Name, provider.URL(newFolder, newName, request.Share))
	} else {
		message = fmt.Sprintf("%s successfully renamed to %s", oldItem.Name, newItem.Name)
	}

	a.rendererApp.Redirect(w, r, fmt.Sprintf("?d=%s", request.Display), renderer.NewSuccessMessage(message))
}

func getNewFolder(r *http.Request, request provider.Request) (string, error) {
	newFolder, err := checkFolderName(r.FormValue("newFolder"), request)
	if err != nil {
		return "", err
	}

	return provider.SanitizeName(newFolder, false)
}

func getNewName(r *http.Request) (string, error) {
	newName, err := checkFormName(r, "newName")
	if err != nil {
		return "", err
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
