package crud

import (
	"context"
	"errors"
	"fmt"
	"mime/multipart"
	"net/http"
	"path"
	"strconv"

	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/httputils/v4/pkg/logger"
	"github.com/ViBiOh/httputils/v4/pkg/model"
	"github.com/ViBiOh/httputils/v4/pkg/renderer"
)

func (a App) saveUploadedFile(ctx context.Context, request provider.Request, inputName, rawSize string, part *multipart.Part) (filename string, err error) {
	var filepath string

	filename, filepath, err = getUploadNameAndPath(request, inputName, part)
	if err != nil {
		return "", fmt.Errorf("unable to get upload name: %s", err)
	}

	var size int64
	size, err = getUploadSize(rawSize)
	if err != nil {
		return "", fmt.Errorf("unable to get upload size: %s", err)
	}

	err = provider.WriteToStorage(ctx, a.storageApp, filepath, size, part)

	if err == nil {
		go func() {
			if info, infoErr := a.storageApp.Info(context.Background(), filepath); infoErr != nil {
				logger.Error("unable to get info for upload event: %s", infoErr)
			} else {
				a.notify(provider.NewUploadEvent(request, info, a.bestSharePath(filepath), a.rendererApp))
			}
		}()
	}

	return filename, err
}

func getUploadNameAndPath(request provider.Request, inputName string, part *multipart.Part) (filename string, filepath string, err error) {
	if !request.Share.IsZero() && request.Share.File {
		return path.Base(request.Share.Path), request.Share.Path, nil
	}

	if len(inputName) != 0 {
		filename = inputName
	} else {
		filename = part.FileName()
	}

	filename, err = provider.SanitizeName(filename, true)
	if err != nil {
		return
	}
	filepath = request.SubPath(filename)

	return
}

func getUploadSize(rawSize string) (int64, error) {
	var size int64 = -1

	if len(rawSize) > 0 {
		if size, err := strconv.ParseInt(rawSize, 10, 64); err != nil {
			return size, fmt.Errorf("unable to parse filesize: %s", err)
		}
	}

	return size, nil
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

	filename, err := a.saveUploadedFile(ctx, request, values["filename"], values["size"], part)
	if err != nil {
		a.error(w, r, request, model.WrapInternal(err))
		return
	}

	var shareID string
	if shared {
		id, err := a.shareApp.Create(ctx, path.Join(request.Path, filename), false, false, "", false, duration)
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
