package crud

import (
	"archive/zip"
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path"

	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/fibr/pkg/thumbnail"
	"github.com/ViBiOh/httputils/v4/pkg/concurrent"
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
func (a App) List(request provider.Request, message renderer.Message) (string, int, map[string]interface{}, error) {
	files, err := a.storageApp.List(request.GetFilepath(""))
	if err != nil {
		return "", 0, nil, err
	}

	items := make([]provider.RenderItem, len(files))
	wg := concurrent.NewLimited(4)

	for index, item := range files {
		func(item provider.StorageItem, index int) {
			wg.Go(func() {
				aggregate, err := a.exifApp.GetAggregateFor(item)
				if err != nil {
					logger.WithField("fn", "crud.List").WithField("item", item.Pathname).Error("unable to read: %s", err)
				}

				items[index] = provider.RenderItem{
					ID:          sha.New(item.Name),
					URL:         request.Item(item),
					Folder:      request.Folder(item),
					StorageItem: item,
					Aggregate:   aggregate,
				}
			})
		}(item, index)
	}

	var hasMap bool
	wg.Go(func() {
		if aggregate, err := a.exifApp.GetAggregateFor(provider.StorageItem{
			IsDir:    true,
			Pathname: request.GetFilepath(""),
		}); err != nil {
			logger.WithField("fn", "crud.List").WithField("item", request.Path).Error("unable to get aggregate: %s", err)
		} else if len(aggregate.Location) != 0 {
			hasMap = true
		}
	})

	wg.Wait()

	content := map[string]interface{}{
		"Paths": getPathParts(request.URL("")),
		"Files": items,
		"Cover": a.getCover(files),

		"Request": request,
		"Message": message,
		"HasMap":  hasMap,
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
		if closeErr := zipWriter.Close(); closeErr != nil {
			logger.Error("unable to close zip: %s", closeErr)
		}
	}()

	filename := path.Base(request.Path)
	if filename == "/" && !request.Share.IsZero() {
		filename = path.Base(path.Join(request.Share.RootName, request.Path))
	}

	w.Header().Add("Content-Disposition", fmt.Sprintf("attachment; filename=%s.zip", filename))

	done := r.Context().Done()
	if err := a.zipFiles(done, request, zipWriter, ""); err != nil {
		select {
		case <-done:
			return
		default:
			a.rendererApp.Error(w, r, err)
		}
	}
}

func (a App) zipFiles(done <-chan struct{}, request provider.Request, zipWriter *zip.Writer, pathname string) error {
	files, err := a.storageApp.List(request.GetFilepath(pathname))
	if err != nil {
		return fmt.Errorf("unable to list: %s", err)
	}

	for _, file := range files {
		select {
		case <-done:
			return errors.New("context is done for zipping files")
		default:
			if file.IsDir {
				if err = a.zipFiles(done, request, zipWriter, path.Join(pathname, file.Name)); err != nil {
					return err
				}
			} else if err = a.addFileToZip(zipWriter, file, pathname); err != nil {
				return err
			}
		}
	}

	return nil
}

func (a App) addFileToZip(zipWriter *zip.Writer, item provider.StorageItem, pathname string) (err error) {
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

	var writer io.Writer
	writer, err = zipWriter.CreateHeader(header)
	if err != nil {
		return fmt.Errorf("unable to create zip header: %s", err)
	}

	var reader io.ReadCloser
	reader, err = a.storageApp.ReaderFrom(item.Pathname)
	if err != nil {
		return fmt.Errorf("unable to read: %w", err)
	}

	defer func() {
		err = provider.HandleClose(reader, err)
	}()

	buffer := provider.BufferPool.Get().(*bytes.Buffer)
	defer provider.BufferPool.Put(buffer)

	_, err = io.CopyBuffer(writer, reader, buffer.Bytes())

	return
}
