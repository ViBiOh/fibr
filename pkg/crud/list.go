package crud

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"path"

	absto "github.com/ViBiOh/absto/pkg/model"
	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/fibr/pkg/thumbnail"
	"github.com/ViBiOh/httputils/v4/pkg/concurrent"
	"github.com/ViBiOh/httputils/v4/pkg/logger"
	"github.com/ViBiOh/httputils/v4/pkg/renderer"
	"go.opentelemetry.io/otel/trace"
)

const (
	uint32max = (1 << 32) - 1
)

func (a App) getCover(ctx context.Context, request provider.Request, files []absto.Item) map[string]any {
	for _, file := range files {
		if a.thumbnailApp.HasThumbnail(ctx, file, thumbnail.SmallSize) {
			return map[string]any{
				"Img":       provider.StorageToRender(file, request),
				"ImgHeight": thumbnail.SmallSize,
				"ImgWidth":  thumbnail.SmallSize,
			}
		}
	}

	return nil
}

// List render directory web view of given dirPath
func (a App) List(ctx context.Context, request provider.Request, message renderer.Message, item absto.Item, files []absto.Item) (renderer.Page, error) {
	if a.tracer != nil {
		var span trace.Span
		ctx, span = a.tracer.Start(ctx, "list")
		defer span.End()
	}

	items := make([]provider.RenderItem, len(files))
	wg := concurrent.NewLimited(6)

	var thumbnails map[string]absto.Item
	wg.Go(func() {
		var err error
		thumbnails, err = a.thumbnailApp.ListDir(ctx, item)
		if err != nil {
			logger.WithField("item", item.Pathname).Error("unable to list thumbnail: %s", err)
			return
		}
	})

	var hasMap bool
	wg.Go(func() {
		if aggregate, err := a.exifApp.GetAggregateFor(ctx, item); err != nil && !absto.IsNotExist(err) {
			logger.WithField("fn", "crud.List").WithField("item", request.Path).Error("unable to get aggregate: %s", err)
		} else if len(aggregate.Location) != 0 {
			hasMap = true
		}
	})

	for index, item := range files {
		func(item absto.Item, index int) {
			wg.Go(func() {
				aggregate, err := a.exifApp.GetAggregateFor(ctx, item)
				if err != nil {
					logger.WithField("fn", "crud.List").WithField("item", item.Pathname).Error("unable to read: %s", err)
				}

				renderItem := provider.StorageToRender(item, request)
				renderItem.Aggregate = aggregate

				items[index] = renderItem
			})
		}(item, index)
	}

	wg.Wait()

	hasThumbnail, hasStory, cover := a.enrichThumbnail(ctx, request, items, thumbnails)

	content := map[string]any{
		"Paths":        getPathParts(request),
		"Files":        items,
		"Cover":        cover,
		"Request":      request,
		"Message":      message,
		"HasMap":       hasMap,
		"HasThumbnail": hasThumbnail,
		"HasStory":     hasStory,
	}

	if request.CanShare {
		content["Shares"] = a.shareApp.List()
	}

	if request.CanWebhook {
		content["Webhooks"] = a.webhookApp.List()
	}

	return renderer.NewPage("files", http.StatusOK, content), nil
}

func (a App) enrichThumbnail(ctx context.Context, request provider.Request, items []provider.RenderItem, thumbnails map[string]absto.Item) (hasThumbnail bool, hasStory bool, cover map[string]any) {
	renderWithThumbnail := request.Display == provider.GridDisplay

	for index, item := range items {
		if _, ok := thumbnails[a.thumbnailApp.Path(item.Item)]; !ok {
			continue
		}

		hasThumbnail = true

		if cover == nil {
			cover = map[string]any{
				"Img":       item,
				"ImgHeight": thumbnail.SmallSize,
				"ImgWidth":  thumbnail.SmallSize,
			}
		}

		if !hasStory {
			hasStory = a.thumbnailApp.HasLargeThumbnail(ctx, item.Item)
		}

		if renderWithThumbnail {
			items[index].HasThumbnail = true
		} else {
			break
		}
	}

	return
}

// Download content of a directory into a streamed zip
func (a App) Download(w http.ResponseWriter, r *http.Request, request provider.Request, items []absto.Item) {
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

	ctx := r.Context()

	if err := a.zipItems(ctx, ctx.Done(), request, zipWriter, items); err != nil {
		a.error(w, r, request, err)
	}
}

func (a App) zipItems(ctx context.Context, done <-chan struct{}, request provider.Request, zipWriter *zip.Writer, items []absto.Item) (err error) {
	for _, item := range items {
		select {
		case <-done:
			logger.Error("context is done for zipping files")
			return nil
		default:
			relativeURL := request.RelativeURL(item)

			if !item.IsDir {
				if err = a.addFileToZip(ctx, zipWriter, item, relativeURL); err != nil {
					return
				}
				continue
			}

			var nestedItems []absto.Item
			nestedItems, err = a.storageApp.List(ctx, request.SubPath(relativeURL))
			if err != nil {
				err = fmt.Errorf("unable to zip nested folder `%s`: %s", relativeURL, err)
				return
			}

			if err = a.zipItems(ctx, done, request, zipWriter, nestedItems); err != nil {
				return err
			}
		}
	}

	return nil
}

func (a App) addFileToZip(ctx context.Context, zipWriter *zip.Writer, item absto.Item, pathname string) (err error) {
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
	reader, err = a.storageApp.ReadFrom(ctx, item.Pathname)
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
