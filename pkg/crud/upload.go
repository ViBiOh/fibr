package crud

import (
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"

	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/httputils/v3/pkg/logger"
)

func (a *app) saveUploadedFile(request provider.Request, part *multipart.Part) (string, error) {
	filename, err := provider.SanitizeName(part.FileName(), true)
	if err != nil {
		return "", err
	}

	filePath := request.GetFilepath(filename)

	hostFile, err := a.storage.WriterTo(filePath)
	if hostFile != nil {
		defer func() {
			if err := hostFile.Close(); err != nil {
				logger.Error("unable to close host file: %s", err)
			}
		}()
	}

	if err != nil {
		return "", err
	}

	copyBuffer := make([]byte, 32*1024)
	if _, err = io.CopyBuffer(hostFile, part, copyBuffer); err != nil {
		return "", err
	}

	info, err := a.storage.Info(filePath)
	if err != nil {
		return "", err
	}

	go a.createThumbnail(info)

	return filename, nil
}

// Upload saves form files to filesystem
func (a *app) Upload(w http.ResponseWriter, r *http.Request, request provider.Request, part *multipart.Part) {
	if !request.CanEdit {
		a.renderer.Error(w, provider.NewError(http.StatusForbidden, ErrNotAuthorized))
		return
	}

	if part == nil {
		a.renderer.Error(w, provider.NewError(http.StatusBadRequest, errors.New("no file provided for save")))
		return
	}

	filename, err := a.saveUploadedFile(request, part)
	if err != nil {
		a.renderer.Error(w, provider.NewError(http.StatusInternalServerError, err))
		return
	}

	content := fmt.Sprintf("File %s successfully uploaded", filename)

	if r.Header.Get("Accept") == "text/plain" {
		w.WriteHeader(http.StatusOK)
		provider.SafeWrite(w, content)

		return
	}

	a.List(w, request, &provider.Message{Level: "success", Content: content})
}
