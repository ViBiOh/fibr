package crud

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"

	absto "github.com/ViBiOh/absto/pkg/model"
	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/httputils/v4/pkg/cntxt"
	"github.com/ViBiOh/httputils/v4/pkg/model"
	"github.com/ViBiOh/httputils/v4/pkg/telemetry"
)

func (s Service) uploadChunk(w http.ResponseWriter, r *http.Request, request provider.Request, fileName, chunkNumber string, file io.Reader) {
	if file == nil {
		s.error(w, r, request, model.WrapInvalid(errors.New("no file provided for save")))
		return
	}

	fileName, err := safeFilename(fileName)
	if err != nil {
		s.error(w, r, request, model.WrapInvalid(err))
		return
	}

	tempDestination := filepath.Join(s.temporaryFolder, provider.Hash(fileName))
	tempFile := filepath.Join(tempDestination, chunkNumber)

	if err = os.MkdirAll(tempDestination, absto.DirectoryPerm); err != nil {
		s.error(w, r, request, model.WrapInternal(err))
		return
	}

	var writer *os.File
	writer, err = os.OpenFile(tempFile, os.O_RDWR|os.O_CREATE|os.O_TRUNC, absto.RegularFilePerm)
	if err != nil {
		return
	}

	defer func() {
		if closeErr := writer.Close(); closeErr != nil {
			slog.LogAttrs(r.Context(), slog.LevelError, "close chunk writer", slog.Any("error", closeErr))
		}

		if err == nil {
			return
		}

		if removeErr := os.Remove(tempFile); removeErr != nil {
			slog.LogAttrs(r.Context(), slog.LevelError, "remove chunk file", slog.String("file", tempFile), slog.Any("error", removeErr))
		}
	}()

	if _, err = io.Copy(writer, file); err != nil {
		s.error(w, r, request, model.WrapInternal(err))
		return
	}

	w.WriteHeader(http.StatusAccepted)
}

func (s Service) mergeChunk(w http.ResponseWriter, r *http.Request, request provider.Request, values map[string]string) {
	ctx, end := telemetry.StartSpan(r.Context(), s.tracer, "merge_chunk")
	defer end(nil)

	fileName, err := safeFilename(values["filename"])
	if err != nil {
		s.error(w, r, request, model.WrapInvalid(err))
		return
	}

	tempFolder := filepath.Join(s.temporaryFolder, provider.Hash(fileName))
	tempFile := filepath.Join(tempFolder, fileName)

	if err := s.mergeChunkFiles(ctx, tempFolder, tempFile); err != nil {
		s.error(w, r, request, model.WrapInternal(err))
		return
	}

	var size int64
	size, err = getUploadSize(values["size"])
	if err != nil {
		s.error(w, r, request, model.WrapInternal(err))
		return
	}

	file, err := os.Open(tempFile)
	if err != nil {
		s.error(w, r, request, model.WrapInternal(err))
		return
	}

	fileName, err = provider.SanitizeName(fileName, true)
	if err != nil {
		s.error(w, r, request, model.WrapInternal(err))
		return
	}

	filePath := request.SubPath(fileName)
	err = provider.WriteToStorage(ctx, s.storage, filePath, size, file)
	if err != nil {
		s.error(w, r, request, model.WrapInternal(err))
		return
	}

	go func(ctx context.Context) {
		if info, infoErr := s.storage.Stat(ctx, filePath); infoErr != nil {
			slog.LogAttrs(ctx, slog.LevelError, "get info for upload event", slog.Any("error", infoErr))
		} else {
			s.pushEvent(ctx, provider.NewUploadEvent(ctx, request, info, s.bestSharePath(filePath), s.renderer))
		}
	}(cntxt.WithoutDeadline(ctx))

	if err = os.RemoveAll(tempFolder); err != nil {
		slog.LogAttrs(ctx, slog.LevelError, "delete chunk folder", slog.String("folder", tempFolder), slog.Any("error", err))
	}

	s.postUpload(w, r, request, fileName)
}

func (s Service) mergeChunkFiles(ctx context.Context, directory, destination string) error {
	writer, err := os.OpenFile(destination, os.O_RDWR|os.O_CREATE|os.O_TRUNC, absto.RegularFilePerm)
	if err != nil {
		return fmt.Errorf("open destination file `%s`: %w", destination, err)
	}

	defer func() {
		if closeErr := writer.Close(); closeErr != nil {
			slog.LogAttrs(ctx, slog.LevelError, "close chunk's destination", slog.Any("error", closeErr))
		}

		if err == nil {
			return
		}

		if removeErr := os.Remove(destination); removeErr != nil {
			slog.LogAttrs(ctx, slog.LevelError, "remove chunk's destination", slog.String("destination", destination), slog.Any("error", removeErr))
		}
	}()

	if err = browseChunkFiles(ctx, directory, destination, writer); err != nil {
		return fmt.Errorf("walk chunks in `%s`: %w", directory, err)
	}

	return nil
}

func browseChunkFiles(ctx context.Context, directory, destination string, writer io.Writer) error {
	return filepath.WalkDir(directory, func(path string, info fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() || path == destination {
			return nil
		}

		reader, err := os.Open(path)
		if err != nil {
			return fmt.Errorf("open chunk `%s`: %w", path, err)
		}

		defer func() {
			if closeErr := reader.Close(); closeErr != nil {
				slog.LogAttrs(ctx, slog.LevelError, "close chunk", slog.String("path", path), slog.Any("error", err))
			}
		}()

		if _, err = io.Copy(writer, reader); err != nil {
			return fmt.Errorf("copy chunk `%s`: %w", path, err)
		}

		return nil
	})
}

func safeFilename(fileName string) (string, error) {
	if err := absto.ValidPath(fileName); err != nil {
		return fileName, err
	}

	output, err := provider.SanitizeName(fileName, true)
	if err != nil {
		return fileName, fmt.Errorf("sanitize: %w", err)
	}

	return output, nil
}
