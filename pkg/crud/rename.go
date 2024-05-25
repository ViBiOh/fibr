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
	"github.com/ViBiOh/httputils/v4/pkg/telemetry"
)

func (s Service) DoRename(ctx context.Context, oldPath, newPath string, oldItem absto.Item) (absto.Item, error) {
	if err := s.storage.Rename(ctx, oldPath, newPath); err != nil {
		return absto.Item{}, fmt.Errorf("rename: %w", err)
	}

	newItem, err := s.storage.Stat(ctx, newPath)
	if err != nil {
		return absto.Item{}, fmt.Errorf("get info of new item: %w", err)
	}

	go s.pushEvent(context.WithoutCancel(ctx), provider.NewRenameEvent(ctx, oldItem, newItem, s.bestSharePath(newPath), s.renderer))

	return newItem, nil
}

func parseRenameParams(r *http.Request, request provider.Request) (string, string, string, string, bool, error) {
	oldName, err := checkFormName(r, "name")
	if err != nil && !errors.Is(err, ErrEmptyName) {
		return "", "", "", "", false, err
	}

	newFolder, err := getNewFolder(r)
	if err != nil {
		return "", "", "", "", false, err
	}

	newName, err := getNewName(r)
	if err != nil {
		return "", "", "", "", false, err
	}

	if strings.HasSuffix(oldName, "/") {
		newName = provider.Dirname(newName)
	}

	cover, err := getFormBool(r.Form.Get("cover"))
	if err != nil {
		return "", "", "", "", false, err
	}

	oldPath := request.SubPath(oldName)
	newPath := provider.GetPathname(newFolder, newName, request.Share)

	return oldPath, newPath, newFolder, newName, cover, nil
}

func (s Service) Rename(w http.ResponseWriter, r *http.Request, request provider.Request) {
	ctx := r.Context()
	telemetry.SetRouteTag(ctx, "/rename")

	if !request.CanEdit {
		s.error(w, r, request, model.WrapForbidden(ErrNotAuthorized))
		return
	}

	oldPath, newPath, newFolder, newName, cover, err := parseRenameParams(r, request)
	if err != nil {
		s.error(w, r, request, err)
		return
	}

	var oldItem absto.Item
	var newItem absto.Item

	if !strings.EqualFold(oldPath, newPath) {
		if _, err := s.checkFile(ctx, newPath, false); err != nil {
			s.error(w, r, request, err)
			return
		}

		oldItem, err = s.checkFile(ctx, oldPath, true)
		if err != nil {
			s.error(w, r, request, err)
			return
		}

		newItem, err = s.DoRename(ctx, oldPath, newPath, oldItem)
		if err != nil {
			s.error(w, r, request, model.WrapInternal(err))
			return
		}

		if oldItem.IsDir() {
			updatePreferences(request, oldPath, newPath)
			provider.SetPrefsCookie(w, request)
		}
	} else {
		newItem, err = s.checkFile(ctx, oldPath, true)
		if err != nil {
			s.error(w, r, request, err)
			return
		}
	}

	if !newItem.IsDir() {
		if cover {
			if err := s.updateCover(ctx, newItem); err != nil {
				s.error(w, r, request, model.WrapInternal(err))
				return
			}
		}

		var tags []string

		if rawTags := r.Form.Get("tags"); len(rawTags) > 0 {
			tags = strings.Split(rawTags, " ")
		}

		if _, err = s.metadata.Update(ctx, newItem, provider.ReplaceTags(tags)); err != nil {
			s.error(w, r, request, model.WrapInternal(err))
			return
		}
	}

	var message string

	if newFolder != request.Path {
		message = fmt.Sprintf("%s successfully moved to %s", oldItem.Name(), provider.URL(newFolder, newName, request.Share))
	} else if oldPath != newPath {
		message = fmt.Sprintf("%s successfully renamed to %s", oldItem.Name(), newItem.Name())
	} else {
		message = fmt.Sprintf("%s successfully updated", newItem.Name())
	}

	s.renderer.Redirect(w, r, fmt.Sprintf("?d=%s", request.Display), renderer.NewSuccessMessage(message))
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

func (s Service) checkFile(ctx context.Context, pathname string, shouldExist bool) (info absto.Item, err error) {
	info, err = s.storage.Stat(ctx, pathname)

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

func (s Service) updateCover(ctx context.Context, item absto.Item) error {
	directory, err := s.storage.Stat(ctx, item.Dir())
	if err != nil {
		return fmt.Errorf("get directory: %w", err)
	}

	aggregate, err := s.metadata.GetAggregateFor(ctx, directory)
	if err != nil && !absto.IsNotExist(err) {
		return fmt.Errorf("get aggregate: %w", err)
	}

	aggregate.Cover = item.Name()

	if err := s.metadata.SaveAggregateFor(ctx, directory, aggregate); err != nil {
		return fmt.Errorf("save aggregate: %w", err)
	}

	return nil
}

func updatePreferences(request provider.Request, oldPath, newPath string) {
	paths := request.Preferences.LayoutPaths

	for path, display := range paths {
		if path == oldPath {
			delete(paths, path)

			paths[newPath] = display
		}
	}
}
