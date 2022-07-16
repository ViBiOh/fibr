package crud

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"

	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/httputils/v4/pkg/logger"
	"github.com/ViBiOh/httputils/v4/pkg/model"
	"github.com/ViBiOh/httputils/v4/pkg/sha"
)

// UploadChunk save chunk file to a temp file
func (a App) UploadChunk(w http.ResponseWriter, r *http.Request, request provider.Request, values map[string]string, file *multipart.Part) {
	var err error

	tempDestination := filepath.Join(temporaryFolder, sha.New(values["filename"]))
	tempFile := filepath.Join(tempDestination, values["chunkNumber"])

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

// MergeChunk merges previously uploaded chunks into one file and move it to final destination
func (a App) MergeChunk(w http.ResponseWriter, r *http.Request, request provider.Request, values map[string]string) {
	var err error

	tempFolder := filepath.Join(temporaryFolder, sha.New(values["filename"]))
	tempFile := filepath.Join(tempFolder, values["filename"])

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

	filePath := request.SubPath(values["filename"])
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

	w.WriteHeader(http.StatusCreated)
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

	if err = filepath.WalkDir(directory, func(path string, _ fs.DirEntry, err error) error {
		if err != nil {
			return err
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
	}); err != nil {
		return fmt.Errorf("unable to walk chunks in `%s`: %s", directory, err)
	}

	return nil
}
