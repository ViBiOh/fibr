package crud

import (
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strings"

	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/httputils/v2/pkg/errors"
	"github.com/ViBiOh/httputils/v2/pkg/logger"
)

const (
	defaultMaxMemory = 128 << 20 // 128 MB
)

var (
	copyBuffer = make([]byte, 32*1024)
)

func (a *app) saveUploadedFile(request *provider.Request, uploadedFile io.ReadCloser, uploadedFileHeader *multipart.FileHeader) (string, error) {
	filename, err := provider.SanitizeName(uploadedFileHeader.Filename, true)
	if err != nil {
		return "", err
	}

	filePath := request.GetFilepath(filename)

	hostFile, err := a.storage.WriterTo(filePath)
	if hostFile != nil {
		defer func() {
			if err := hostFile.Close(); err != nil {
				logger.Error("%#v", err)
			}
		}()
	}

	if err != nil {
		return "", err
	}

	if _, err = io.CopyBuffer(hostFile, uploadedFile, copyBuffer); err != nil {
		return "", errors.WithStack(err)
	}

	info, err := a.storage.Info(filePath)
	if err != nil {
		return "", err
	}

	go a.createThumbnail(info)

	return filename, nil
}

// Upload saves form files to filesystem
func (a *app) Upload(w http.ResponseWriter, r *http.Request, request *provider.Request) {
	if !request.CanEdit {
		a.renderer.Error(w, provider.NewError(http.StatusForbidden, ErrNotAuthorized))
		return
	}

	if err := r.ParseMultipartForm(defaultMaxMemory); err != nil {
		a.renderer.Error(w, provider.NewError(http.StatusBadRequest, errors.WithStack(err)))
		return
	}

	if r.MultipartForm.File == nil || len(r.MultipartForm.File["files[]"]) == 0 {
		a.renderer.Error(w, provider.NewError(http.StatusBadRequest, errors.New("no file provided for save")))
		return
	}

	filenames := make([]string, len(r.MultipartForm.File["files[]"]))

	for index, file := range r.MultipartForm.File["files[]"] {
		uploadedFile, err := file.Open()
		if uploadedFile != nil {
			defer func() {
				if err := uploadedFile.Close(); err != nil {
					logger.Error("%#v", errors.WithStack(err))
				}
			}()
		}

		if err != nil {
			a.renderer.Error(w, provider.NewError(http.StatusBadRequest, errors.WithStack(err)))
			return
		}

		filename, err := a.saveUploadedFile(request, uploadedFile, file)
		if err != nil {
			a.renderer.Error(w, provider.NewError(http.StatusInternalServerError, err))
			return
		}

		filenames[index] = filename
	}

	message := fmt.Sprintf("File %s successfully uploaded", filenames[0])
	if len(filenames) > 1 {
		message = fmt.Sprintf("Files %s successfully uploaded", strings.Join(filenames, ", "))
	}

	a.List(w, request, &provider.Message{Level: "success", Content: message})
}
