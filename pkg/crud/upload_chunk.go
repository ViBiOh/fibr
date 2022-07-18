package crud

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"

	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/httputils/v4/pkg/logger"
	"github.com/ViBiOh/httputils/v4/pkg/model"
	"github.com/ViBiOh/httputils/v4/pkg/sha"
	"github.com/ViBiOh/httputils/v4/pkg/tracer"
)

func (a App) uploadChunk(w http.ResponseWriter, r *http.Request, request provider.Request, fileName, chunkNumber string, file io.Reader) {
	if file == nil {
		a.error(w, r, request, model.WrapInvalid(errors.New("no file provided for save")))
		return
	}

	var err error

	tempDestination := filepath.Join(a.temporaryFolder, sha.New(fileName))
	tempFile := filepath.Join(tempDestination, chunkNumber)

	if err = os.MkdirAll(tempDestination, 0o700); err != nil {
		a.error(w, r, request, model.WrapInternal(err))
		return
	}

	var writer *os.File
	writer, err = os.OpenFile(tempFile, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return
	}

	defer func() {
		if closeErr := writer.Close(); closeErr != nil {
			logger.Error("unable to close chunk writer: %s", closeErr)
		}

		if err == nil {
			return
		}

		if removeErr := os.Remove(tempFile); removeErr != nil {
			logger.Error("unable to remove chunk file `%s`: %s", tempFile, removeErr)
		}
	}()

	if _, err = io.Copy(writer, file); err != nil {
		a.error(w, r, request, model.WrapInternal(err))
		return
	}

	w.WriteHeader(http.StatusAccepted)
}

func (a App) mergeChunk(w http.ResponseWriter, r *http.Request, request provider.Request, values map[string]string) {
	ctx, end := tracer.StartSpan(r.Context(), a.tracer, "mergeChunk")
	defer end()

	var err error

	fileName := values["filename"]
	tempFolder := filepath.Join(a.temporaryFolder, sha.New(fileName))
	tempFile := filepath.Join(tempFolder, fileName)

	if err := a.mergeChunkFiles(tempFolder, tempFile); err != nil {
		a.error(w, r, request, model.WrapInternal(err))
		return
	}

	var size int64
	size, err = getUploadSize(values["size"])
	if err != nil {
		a.error(w, r, request, model.WrapInternal(err))
		return
	}

	file, err := os.Open(tempFile)
	if err != nil {
		a.error(w, r, request, model.WrapInternal(err))
		return
	}

	fileName, err = provider.SanitizeName(fileName, true)
	if err != nil {
		a.error(w, r, request, model.WrapInternal(err))
		return
	}
	filePath := request.SubPath(fileName)
	err = provider.WriteToStorage(r.Context(), a.storageApp, filePath, size, file)

	if err == nil {
		go func() {
			if info, infoErr := a.storageApp.Info(context.Background(), filePath); infoErr != nil {
				logger.Error("unable to get info for upload event: %s", infoErr)
			} else {
				a.notify(provider.NewUploadEvent(request, info, a.bestSharePath(filePath), a.rendererApp))
			}
		}()
	}

	if err = os.RemoveAll(tempFolder); err != nil {
		logger.Error("unable to delete chunk folder `%s`: %s", tempFolder, err)
	}

	a.postUpload(ctx, w, r, request, fileName, values)
}

func (a App) mergeChunkFiles(directory, destination string) error {
	writer, err := os.OpenFile(destination, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return fmt.Errorf("unable to open destination file `%s`: %s", destination, err)
	}

	defer func() {
		if closeErr := writer.Close(); closeErr != nil {
			logger.Error("unable to close chunk's destination: %s", closeErr)
		}

		if err == nil {
			return
		}

		if removeErr := os.Remove(destination); removeErr != nil {
			logger.Error("unable to remove chunk's destination `%s`: %s", destination, removeErr)
		}
	}()

	if err = browseChunkFiles(directory, destination, writer); err != nil {
		return fmt.Errorf("unable to walk chunks in `%s`: %s", directory, err)
	}

	return nil
}

func browseChunkFiles(directory, destination string, writer io.Writer) error {
	return filepath.WalkDir(directory, func(path string, info fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() || path == destination {
			return nil
		}

		reader, err := os.Open(path)
		if err != nil {
			return fmt.Errorf("unable to open chunk `%s`: %s", path, err)
		}

		defer func() {
			if closeErr := reader.Close(); closeErr != nil {
				logger.Error("unable to close chunk `%s`: %s", path, err)
			}
		}()

		if _, err = io.Copy(writer, reader); err != nil {
			return fmt.Errorf("unable to copy chunk `%s`: %s", path, err)
		}

		return nil
	})
}
