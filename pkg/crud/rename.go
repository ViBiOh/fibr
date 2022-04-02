package crud

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	absto "github.com/ViBiOh/absto/pkg/model"
	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/httputils/v4/pkg/model"
	"github.com/ViBiOh/httputils/v4/pkg/renderer"
)

func (a App) doRename(ctx context.Context, oldPath, newPath string, oldItem absto.Item) (absto.Item, error) {
	if err := a.storageApp.Rename(ctx, oldPath, newPath); err != nil {
		return absto.Item{}, fmt.Errorf("unable to rename: %w", err)
	}

	newItem, err := a.storageApp.Info(ctx, newPath)
	if err != nil {
		return absto.Item{}, fmt.Errorf("unable to get info of new item: %w", err)
	}

	go a.notify(provider.NewRenameEvent(oldItem, newItem, a.bestSharePath(newPath), a.rendererApp))

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
	ctx := r.Context()

	if _, err := a.checkFile(ctx, newPath, false); err != nil {
		a.error(w, r, request, err)
		return
	}

	oldItem, err := a.checkFile(ctx, oldPath, true)
	if err != nil {
		a.error(w, r, request, err)
		return
	}

	newItem, err := a.doRename(ctx, oldPath, newPath, oldItem)
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

func (a App) checkFile(ctx context.Context, pathname string, shouldExist bool) (info absto.Item, err error) {
	info, err = a.storageApp.Info(ctx, pathname)

	if err == nil {
		if !shouldExist {
			err = model.WrapInvalid(fmt.Errorf("`%s` already exist", pathname))
		}
	} else {
		if !absto.IsNotExist(err) {
			err = model.WrapInternal(err)
		} else if shouldExist {
			err = model.WrapNotFound(fmt.Errorf("`%s` not found", pathname))
		} else {
			err = nil
		}
	}

	return
}

func updatePreferences(request provider.Request, oldPath, newPath string) {
	paths := request.Preferences.LayoutPaths

	for index, layoutPath := range paths {
		if layoutPath != oldPath {
			paths[index] = newPath
		} else {
			paths[index] = layoutPath
		}
	}
}
