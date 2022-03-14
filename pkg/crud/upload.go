package crud

import (
	"context"
	"errors"
	"fmt"
	"mime/multipart"
	"net/http"
	"path"

	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/httputils/v4/pkg/logger"
	"github.com/ViBiOh/httputils/v4/pkg/model"
	"github.com/ViBiOh/httputils/v4/pkg/renderer"
)

func (a App) saveUploadedFile(ctx context.Context, request provider.Request, inputName string, part *multipart.Part) (filename string, err error) {
	var filePath string

	if !request.Share.IsZero() && request.Share.File {
		filename = path.Base(request.Share.Path)
		filePath = request.Share.Path
	} else {
		if len(inputName) != 0 {
			filename = inputName
		} else {
			filename = part.FileName()
		}

		filename, err = provider.SanitizeName(filename, true)
		if err != nil {
			return "", err
		}
		filePath = request.SubPath(filename)
	}

	err = provider.WriteToStorage(ctx, a.storageApp, filePath, part)

	if err == nil {
		go func() {
			if info, infoErr := a.storageApp.Info(context.Background(), filePath); infoErr != nil {
				logger.Error("unable to get info for upload event: %s", infoErr)
			} else {
				a.notify(provider.NewUploadEvent(request, info, a.bestSharePath(filePath), a.rendererApp))
			}
		}()
	}

	return filename, err
}

// Upload saves form files to filesystem
func (a App) Upload(w http.ResponseWriter, r *http.Request, request provider.Request, values map[string]string, part *multipart.Part) {
	if !request.CanEdit {
		a.error(w, r, request, model.WrapForbidden(ErrNotAuthorized))
		return
	}

	if part == nil {
		a.error(w, r, request, model.WrapInvalid(errors.New("no file provided for save")))
		return
	}

	shared, err := getFormBool(values["share"])
	if err != nil {
		a.error(w, r, request, model.WrapInvalid(err))
		return
	}

	duration, err := getFormDuration(values["duration"])
	if err != nil {
		a.error(w, r, request, model.WrapInvalid(err))
		return
	}

	ctx := r.Context()

	filename, err := a.saveUploadedFile(ctx, request, values["filename"], part)
	if err != nil {
		a.error(w, r, request, model.WrapInternal(err))
		return
	}

	var shareID string
	if shared {
		id, err := a.shareApp.Create(ctx, path.Join(request.Path, filename), false, "", false, duration)
		if err != nil {
			a.error(w, r, request, model.WrapInternal(err))
			return
		}

		shareID = id
	}

	if r.Header.Get("Accept") == "text/plain" {
		w.WriteHeader(http.StatusCreated)
		provider.SafeWrite(w, filename)
		if shared {
			provider.SafeWrite(w, fmt.Sprintf("\n%s", shareID))
		}

		return
	}

	message := fmt.Sprintf("File %s successfully uploaded", filename)
	if shared {
		message = fmt.Sprintf("%s. Share ID is %s", message, shareID)
	}

	a.rendererApp.Redirect(w, r, fmt.Sprintf("?d=%s", request.Display), renderer.NewSuccessMessage(message))
}
