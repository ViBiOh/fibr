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
	"github.com/ViBiOh/httputils/v4/pkg/concurrent"
	"github.com/ViBiOh/httputils/v4/pkg/logger"
	"github.com/ViBiOh/httputils/v4/pkg/renderer"
)

const (
	uint32max = (1 << 32) - 1
)

func (a App) getCover(files []provider.StorageItem) map[string]interface{} {
	for _, file := range files {
		if file.IsDir || file.IsVideo() {
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
func (a App) List(request provider.Request, message renderer.Message, files []provider.StorageItem) (string, int, map[string]interface{}, error) {
	items := make([]provider.RenderItem, len(files))
	wg := concurrent.NewLimited(4)

	for index, item := range files {
		func(item provider.StorageItem, index int) {
			wg.Go(func() {
				aggregate, err := a.exifApp.GetAggregateFor(item)
				if err != nil {
					logger.WithField("fn", "crud.List").WithField("item", item.Pathname).Error("unable to read: %s", err)
				}

				render := provider.StorageToRender(item, request)
				render.Aggregate = aggregate
				items[index] = render
			})
		}(item, index)
	}

	var hasMap bool
	wg.Go(func() {
		if aggregate, err := a.exifApp.GetAggregateFor(provider.StorageItem{
			IsDir:    true,
			Pathname: request.Filepath(),
		}); err != nil {
			logger.WithField("fn", "crud.List").WithField("item", request.Path).Error("unable to get aggregate: %s", err)
		} else if len(aggregate.Location) != 0 {
			hasMap = true
		}
	})

	wg.Wait()

	content := map[string]interface{}{
		"Paths":   getPathParts(request),
		"Files":   items,
		"Cover":   a.getCover(files),
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
func (a App) Download(w http.ResponseWriter, r *http.Request, request provider.Request, items []provider.StorageItem) {
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

	if err := a.zipItems(r.Context().Done(), request, zipWriter, items); err != nil {
		a.rendererApp.Error(w, r, err)
	}
}

func (a App) zipItems(done <-chan struct{}, request provider.Request, zipWriter *zip.Writer, items []provider.StorageItem) (err error) {
	for _, item := range items {
		select {
		case <-done:
			logger.Error("context is done for zipping files")
			return nil
		default:
			relativeURL := request.RelativeURL(item)

			if !item.IsDir {
				if err = a.addFileToZip(zipWriter, item, relativeURL); err != nil {
					return
				}
				continue
			}

			var nestedItems []provider.StorageItem
			nestedItems, err = a.storageApp.List(request.SubPath(relativeURL))
			if err != nil {
				err = fmt.Errorf("unable to zip nested folder `%s`: %s", relativeURL, err)
				return
			}

			if err = a.zipItems(done, request, zipWriter, nestedItems); err != nil {
				return err
			}
		}
	}

	return nil
}

func (a App) addFileToZip(zipWriter *zip.Writer, item provider.StorageItem, pathname string) (err error) {
	header := &zip.FileHeader{
		Name:               pathname,
		UncompressedSize64: uint64(item.Size),
		UncompressedSize:   uint32(item.Size),
		Modified:           item.Date,
		Method:             zip.Deflate,
	}
	header.SetMode(0o600)

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
