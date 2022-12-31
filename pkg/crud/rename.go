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
	"github.com/ViBiOh/httputils/v4/pkg/tracer"
)

func (a App) doRename(ctx context.Context, oldPath, newPath string, oldItem absto.Item) (absto.Item, error) {
	if err := a.storageApp.Rename(ctx, oldPath, newPath); err != nil {
		return absto.Item{}, fmt.Errorf("rename: %w", err)
	}

	newItem, err := a.storageApp.Info(ctx, newPath)
	if err != nil {
		return absto.Item{}, fmt.Errorf("get info of new item: %w", err)
	}

	go a.notify(tracer.CopyToBackground(ctx), provider.NewRenameEvent(oldItem, newItem, a.bestSharePath(newPath), a.rendererApp))

	return newItem, nil
}

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

	newFolder, err := getNewFolder(r)
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

	cover, err := getFormBool(r.Form.Get("cover"))
	if err != nil {
		a.error(w, r, request, err)
		return
	}

	oldPath := request.SubPath(oldName)
	newPath := provider.GetPathname(newFolder, newName, request.Share)
	ctx := r.Context()

	var oldItem absto.Item
	var newItem absto.Item

	if !strings.EqualFold(oldPath, newPath) {
		if _, err := a.checkFile(ctx, newPath, false); err != nil {
			a.error(w, r, request, err)
			return
		}

		oldItem, err = a.checkFile(ctx, oldPath, true)
		if err != nil {
			a.error(w, r, request, err)
			return
		}

		newItem, err = a.doRename(ctx, oldPath, newPath, oldItem)
		if err != nil {
			a.error(w, r, request, model.WrapInternal(err))
			return
		}

		if oldItem.IsDir {
			updatePreferences(request, oldPath, newPath)
			provider.SetPrefsCookie(w, request)
		}
	} else {
		newItem, err = a.checkFile(ctx, oldPath, true)
		if err != nil {
			a.error(w, r, request, err)
			return
		}
	}

	if !newItem.IsDir {
		if cover {
			if err := a.updateCover(ctx, newItem); err != nil {
				a.error(w, r, request, model.WrapInternal(err))
				return
			}
		}

		if _, err = a.metadataApp.Update(ctx, newItem, provider.ReplaceTags(strings.Split(r.Form.Get("tags"), " "))); err != nil {
			a.error(w, r, request, model.WrapInternal(err))
			return
		}
	}

	var message string

	if newFolder != request.Path {
		message = fmt.Sprintf("%s successfully moved to %s", oldItem.Name, provider.URL(newFolder, newName, request.Share))
	} else if oldPath != newPath {
		message = fmt.Sprintf("%s successfully renamed to %s", oldItem.Name, newItem.Name)
	} else {
		message = fmt.Sprintf("%s successfully updated", newItem.Name)
	}

	a.rendererApp.Redirect(w, r, fmt.Sprintf("?d=%s", request.Display), renderer.NewSuccessMessage(message))
}

func getNewFolder(r *http.Request) (string, error) {
	newFolder, err := checkFolderName(r.FormValue("newFolder"))
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

func (a App) updateCover(ctx context.Context, item absto.Item) error {
	directory, err := a.storageApp.Info(ctx, item.Dir())
	if err != nil {
		return fmt.Errorf("get directory: %w", err)
	}

	aggregate, err := a.metadataApp.GetAggregateFor(ctx, directory)
	if err != nil && !absto.IsNotExist(err) {
		return fmt.Errorf("get aggregate: %w", err)
	}

	aggregate.Cover = item.Name

	if err := a.metadataApp.SaveAggregateFor(ctx, directory, aggregate); err != nil {
		return fmt.Errorf("save aggregate: %w", err)
	}

	return nil
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
