package crud

import (
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"path"

	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/fibr/pkg/thumbnail"
	"github.com/ViBiOh/httputils/v4/pkg/logger"
	"github.com/ViBiOh/httputils/v4/pkg/renderer"
)

func (a *app) saveUploadedFile(request provider.Request, part *multipart.Part) (filename string, err error) {
	var filePath string

	if len(request.Share.ID) != 0 && request.Share.File {
		filename = path.Base(request.Share.Path)
		filePath = request.Share.Path
	} else {
		filename, err = provider.SanitizeName(part.FileName(), true)
		if err != nil {
			return "", err
		}
		filePath = request.GetFilepath(filename)
	}

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

	if thumbnail.CanHaveThumbnail(info) {
		a.thumbnail.GenerateThumbnail(info)
	}

	return filename, nil
}

// Upload saves form files to filesystem
func (a *app) Upload(w http.ResponseWriter, r *http.Request, request provider.Request, part *multipart.Part) {
	if !request.CanEdit {
		a.renderer.Error(w, request, provider.NewError(http.StatusForbidden, ErrNotAuthorized))
		return
	}

	if part == nil {
		a.renderer.Error(w, request, provider.NewError(http.StatusBadRequest, errors.New("no file provided for save")))
		return
	}

	filename, err := a.saveUploadedFile(request, part)
	if err != nil {
		a.renderer.Error(w, request, provider.NewError(http.StatusInternalServerError, err))
		return
	}

	content := fmt.Sprintf("File %s successfully uploaded", filename)

	if r.Header.Get("Accept") == "text/plain" {
		w.WriteHeader(http.StatusOK)
		provider.SafeWrite(w, content)

		return
	}

	http.Redirect(w, r, fmt.Sprintf("%s/?%s", request.GetURI(""), renderer.NewSuccessMessage(content)), http.StatusFound)
}
