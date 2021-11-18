package crud

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"net/http"
	"path"

	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/fibr/pkg/thumbnail"
	"github.com/ViBiOh/httputils/v4/pkg/logger"
	"github.com/ViBiOh/httputils/v4/pkg/renderer"
	"github.com/ViBiOh/httputils/v4/pkg/sha"
)

const (
	uint32max = (1 << 32) - 1
)

func (a App) getCover(files []provider.StorageItem) map[string]interface{} {
	for _, file := range files {
		if file.IsVideo() {
			continue
		}

		if a.thumbnailApp.HasThumbnail(file) {
			return map[string]interface{}{
				"Img":       file,
				"ImgHeight": thumbnail.Height,
				"ImgWidth":  thumbnail.Width,
			}
		}
	}

	return nil
}

// List render directory web view of given dirPath
func (a App) List(w http.ResponseWriter, request provider.Request, message renderer.Message) (string, int, map[string]interface{}, error) {
	files, err := a.storageApp.List(request.GetFilepath(""))
	if err != nil {
		return "", 0, nil, err
	}

	uri := request.URL("")

	items := make([]provider.RenderItem, len(files))
	for index, item := range files {
		aggregate, err := a.exifApp.GetAggregateFor(item)
		if err != nil {
			logger.WithField("fn", "crud.List").WithField("item", item.Pathname).Error("unable to read: %s", err)
		}

		items[index] = provider.RenderItem{
			ID:          sha.New(item.Name),
			URI:         uri,
			StorageItem: item,
			Aggregate:   aggregate,
		}
	}

	content := map[string]interface{}{
		"Paths": getPathParts(uri),
		"Files": items,
		"Cover": a.getCover(files),

		"Request": request,
		"Message": message,
	}

	if request.CanShare {
		content["Shares"] = a.shareApp.List()
	}

	if request.CanWebhook {
		content["Webhooks"] = a.webhookApp.List()
	}

	return "files", http.StatusOK, content, nil
}

// Download content of a directory into a streamed zip
func (a App) Download(w http.ResponseWriter, r *http.Request, request provider.Request) {
	zipWriter := zip.NewWriter(w)
	defer func() {
		if err := zipWriter.Close(); err != nil {
			logger.Error("unable to close zip: %s", err)
		}
	}()

	filename := path.Base(request.Path)
	if filename == "/" && len(request.Share.ID) != 0 {
		filename = path.Base(path.Join(request.Share.RootName, request.Path))
	}

	w.Header().Add("Content-Disposition", fmt.Sprintf("attachment; filename=%s.zip", filename))

	if err := a.zipFiles(request, zipWriter, ""); err != nil {
		a.rendererApp.Error(w, r, err)
	}
}

func (a App) zipFiles(request provider.Request, zipWriter *zip.Writer, pathname string) error {
	files, err := a.storageApp.List(request.GetFilepath(pathname))
	if err != nil {
		return err
	}

	for _, file := range files {
		if file.IsDir {
			if err := a.zipFiles(request, zipWriter, path.Join(pathname, file.Name)); err != nil {
				return err
			}
		} else if err := a.addFileToZip(zipWriter, file, pathname); err != nil {
			return err
		}
	}

	return nil
}

func (a App) addFileToZip(zipWriter *zip.Writer, item provider.StorageItem, pathname string) error {
	header := &zip.FileHeader{
		Name:               path.Join(pathname, item.Name),
		UncompressedSize64: uint64(item.Size),
		UncompressedSize:   uint32(item.Size),
		Modified:           item.Date,
		Method:             zip.Deflate,
	}

	if item.IsDir {
		header.SetMode(0o700)
	} else {
		header.SetMode(0o600)
	}

	if header.UncompressedSize64 > uint32max {
		header.UncompressedSize = uint32max
	} else {
		header.UncompressedSize = uint32(header.UncompressedSize64)
	}

	writer, err := zipWriter.CreateHeader(header)
	if err != nil {
		return fmt.Errorf("unable to create zip header: %s", err)
	}

	reader, err := a.storageApp.ReaderFrom(item.Pathname)
	if err != nil {
		return fmt.Errorf("unable to read: %w", err)
	}

	buffer := provider.BufferPool.Get().(*bytes.Buffer)
	defer provider.BufferPool.Put(buffer)

	_, err = io.CopyBuffer(writer, reader, buffer.Bytes())

	if closeErr := reader.Close(); closeErr != nil {
		if err != nil {
			err = fmt.Errorf("%s: %w", err, closeErr)
		} else {
			err = fmt.Errorf("unable to close: %s", closeErr)
		}
	}

	return err
}
