package crud

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"mime/multipart"
	"net/http"
	"path"
	"strconv"

	absto "github.com/ViBiOh/absto/pkg/model"
	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/httputils/v4/pkg/model"
	"github.com/ViBiOh/httputils/v4/pkg/renderer"
	"github.com/ViBiOh/httputils/v4/pkg/telemetry"
)

func (s Service) saveUploadedFile(ctx context.Context, request provider.Request, filePath string, size int64, file *multipart.Part) error {
	err := provider.WriteToStorage(ctx, s.storage, filePath, size, file)

	if err == nil {
		go func(ctx context.Context) {
			if info, infoErr := s.storage.Stat(ctx, filePath); infoErr != nil {
				slog.LogAttrs(ctx, slog.LevelError, "get info for upload event", slog.Any("error", infoErr))
			} else {
				s.pushEvent(ctx, provider.NewUploadEvent(ctx, request, info, s.bestSharePath(filePath), s.renderer))
			}
		}(context.WithoutCancel(ctx))
	}

	return err
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

	if err = absto.ValidPath(fileName); err != nil {
		return
	}

	if fileName, err = provider.SanitizeName(fileName, true); err != nil {
		return
	}

	filePath = request.SubPath(fileName)

	return
}

func getUploadSize(rawSize string) (int64, error) {
	var size int64 = -1

	if len(rawSize) > 0 {
		if size, err := strconv.ParseInt(rawSize, 10, 64); err != nil {
			return size, fmt.Errorf("parse filesize: %w", err)
		}
	}

	return size, nil
}

func (s Service) upload(w http.ResponseWriter, r *http.Request, request provider.Request, fileName, filePath string, size int64, file *multipart.Part) {
	if file == nil {
		s.error(w, r, request, model.WrapInvalid(errors.New("no file provided for save")))
		return
	}

	var err error

	ctx, end := telemetry.StartSpan(r.Context(), s.tracer, "upload")
	defer end(&err)

	if err = s.saveUploadedFile(ctx, request, filePath, size, file); err != nil {
		s.error(w, r, request, model.WrapInternal(err))
		return
	}

	s.postUpload(w, r, request, fileName)
}

func (s Service) postUpload(w http.ResponseWriter, r *http.Request, request provider.Request, fileName string) {
	ctx := r.Context()

	if r.Header.Get("Accept") == "text/plain" {
		w.WriteHeader(http.StatusCreated)
		provider.SafeWrite(ctx, w, fileName)

		return
	}

	s.renderer.Redirect(w, r, fmt.Sprintf("?d=%s", request.Display), renderer.NewSuccessMessage("File %s successfully uploaded", fileName))
}
