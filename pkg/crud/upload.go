package crud

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"path"
	"time"

	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/httputils/v4/pkg/logger"
	"github.com/ViBiOh/httputils/v4/pkg/model"
	"github.com/ViBiOh/httputils/v4/pkg/renderer"
)

func (a App) saveUploadedFile(request provider.Request, part *multipart.Part) (filename string, err error) {
	var filePath string

	if !request.Share.IsZero() && request.Share.File {
		filename = path.Base(request.Share.Path)
		filePath = request.Share.Path
	} else {
		filename, err = provider.SanitizeName(part.FileName(), true)
		if err != nil {
			return "", err
		}
		filePath = request.SubPath(filename)
	}

	var writer io.WriteCloser
	writer, err = a.storageApp.WriterTo(filePath)
	if err != nil {
		return "", fmt.Errorf("unable to get writer: %w", err)
	}

	defer func() {
		if err == nil {
			return
		}

		if removeErr := a.storageApp.Remove(filePath); removeErr != nil {
			err = fmt.Errorf("%s: %w", err, removeErr)
		}
	}()

	defer func() {
		err = provider.HandleClose(writer, err)
	}()

	buffer := provider.BufferPool.Get().(*bytes.Buffer)
	defer provider.BufferPool.Put(buffer)

	if _, err = io.CopyBuffer(writer, part, buffer.Bytes()); err != nil {
		return filename, fmt.Errorf("unable to copy: %s", err)
	}

	go func() {
		// Waiting one second for filesystem cache refresh (S3 notably)
		time.Sleep(time.Second)

		if info, infoErr := a.storageApp.Info(filePath); infoErr != nil {
			logger.Error("unable to get info for upload event: %s", infoErr)
		} else {
			a.notify(provider.NewUploadEvent(request, info, a.bestSharePath(request, filename), a.rendererApp))
		}
	}()

	return filename, nil
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

	filename, err := a.saveUploadedFile(request, part)
	if err != nil {
		a.error(w, r, request, model.WrapInternal(err))
		return
	}

	var shareID string
	if shared {
		id, err := a.shareApp.Create(path.Join(request.Path, filename), false, "", false, duration)
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
