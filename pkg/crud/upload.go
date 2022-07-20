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
	"github.com/ViBiOh/httputils/v4/pkg/tracer"
)

func (a App) saveUploadedFile(ctx context.Context, request provider.Request, inputName, rawSize string, file *multipart.Part) (fileName string, err error) {
	var filePath string

	fileName, filePath, err = getUploadNameAndPath(request, inputName, file)
	if err != nil {
		return "", fmt.Errorf("unable to get upload name: %s", err)
	}

	var size int64
	size, err = getUploadSize(rawSize)
	if err != nil {
		return "", fmt.Errorf("unable to get upload size: %s", err)
	}

	err = provider.WriteToStorage(ctx, a.storageApp, filePath, size, file)

	if err == nil {
		go func() {
			if info, infoErr := a.storageApp.Info(context.Background(), filePath); infoErr != nil {
				logger.Error("unable to get info for upload event: %s", infoErr)
			} else {
				a.notify(provider.NewUploadEvent(request, info, a.bestSharePath(filePath), a.rendererApp))
			}
		}()
	}

	return fileName, err
}

func getUploadNameAndPath(request provider.Request, inputName string, part *multipart.Part) (fileName string, filePath string, err error) {
	if !request.Share.IsZero() && request.Share.File {
		return path.Base(request.Share.Path), request.Share.Path, nil
	}

	if len(inputName) != 0 {
		fileName = inputName
	} else {
		fileName = part.FileName()
	}

	fileName, err = provider.SanitizeName(fileName, true)
	if err != nil {
		return
	}
	filePath = request.SubPath(fileName)

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

func (a App) upload(w http.ResponseWriter, r *http.Request, request provider.Request, values map[string]string, file *multipart.Part) {
	if file == nil {
		a.error(w, r, request, model.WrapInvalid(errors.New("no file provided for save")))
		return
	}

	ctx, end := tracer.StartSpan(r.Context(), a.tracer, "upload")
	defer end()

	filename, err := a.saveUploadedFile(ctx, request, values["filename"], values["size"], file)
	if err != nil {
		a.error(w, r, request, model.WrapInternal(err))
		return
	}

	a.postUpload(ctx, w, r, request, filename, values)
}

func (a App) postUpload(ctx context.Context, w http.ResponseWriter, r *http.Request, request provider.Request, fileName string, values map[string]string) {
	if r.Header.Get("Accept") == "text/plain" {
		w.WriteHeader(http.StatusCreated)
		provider.SafeWrite(w, fileName)

		return
	}

	a.rendererApp.Redirect(w, r, fmt.Sprintf("?d=%s", request.Display), renderer.NewSuccessMessage("File %s successfully uploaded", fileName))
}
