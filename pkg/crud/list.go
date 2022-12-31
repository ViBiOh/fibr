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
	"github.com/ViBiOh/fibr/pkg/search"
	"github.com/ViBiOh/fibr/pkg/thumbnail"
	"github.com/ViBiOh/httputils/v4/pkg/concurrent"
	"github.com/ViBiOh/httputils/v4/pkg/logger"
	"github.com/ViBiOh/httputils/v4/pkg/renderer"
	"github.com/ViBiOh/httputils/v4/pkg/tracer"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

func listLogger(pathname string) logger.Provider {
	return logger.WithField("fn", "crud.list").WithField("item", pathname)
}

func (a App) list(ctx context.Context, request provider.Request, message renderer.Message, item absto.Item, files []absto.Item) (renderer.Page, error) {
	ctx, end := tracer.StartSpan(ctx, a.tracer, "list", trace.WithAttributes(attribute.String("item", item.Pathname)))
	defer end()

	wg := concurrent.NewSimple()

	var directoryAggregate provider.Aggregate
	wg.Go(func() {
		var err error

		directoryAggregate, err = a.metadataApp.GetAggregateFor(ctx, item)
		if err != nil && !absto.IsNotExist(err) {
			listLogger(item.Pathname).Error("get aggregate: %s", err)
		}
	})

	var aggregates map[string]provider.Aggregate
	wg.Go(func() {
		var err error

		aggregates, err = a.metadataApp.GetAllAggregateFor(ctx, files...)
		if err != nil {
			listLogger(item.Pathname).Error("list aggregates: %s", err)
		}
	})

	var metadatas map[string]provider.Metadata
	wg.Go(func() {
		var err error

		metadatas, err = a.metadataApp.GetAllMetadataFor(ctx, files...)
		if err != nil {
			listLogger(item.Pathname).Error("list metadatas: %s", err)
		}
	})

	var thumbnails map[string]absto.Item
	thumbnailDone := make(chan struct{})
	go func() {
		defer close(thumbnailDone)

		var err error

		thumbnails, err = a.thumbnailApp.ListDir(ctx, item)
		if err != nil {
			listLogger(item.Pathname).Error("list thumbnail: %s", err)
			return
		}
	}()

	var savedSearches search.Searches
	savedSearchDone := make(chan struct{})
	go func() {
		defer close(savedSearchDone)

		var err error

		savedSearches, err = a.searchApp.List(ctx, item)
		if err != nil {
			listLogger(item.Pathname).Error("list saved searches: %s", err)
			return
		}
	}()

	wg.Wait()

	items := make([]provider.RenderItem, len(files))

	for index, item := range files {
		renderItem := provider.StorageToRender(item, request)
		renderItem.Tags = metadatas[item.ID].Tags

		if item.IsDir {
			renderItem.Aggregate = aggregates[item.ID]
		} else {
			renderItem.IsCover = item.Name == directoryAggregate.Cover
		}

		items[index] = renderItem
	}

	<-thumbnailDone
	hasThumbnail, hasStory, cover := a.enrichThumbnail(ctx, directoryAggregate, items, thumbnails)

	<-savedSearchDone

	content := map[string]any{
		"Paths":         getPathParts(request),
		"Files":         items,
		"SavedSearches": savedSearches,
		"Cover":         cover,
		"Request":       request,
		"Message":       message,
		"HasMap":        len(directoryAggregate.Location),
		"HasThumbnail":  hasThumbnail,
		"HasStory":      hasStory,
		"ChunkUpload":   a.chunkUpload,
	}

	if request.CanShare {
		content["Shares"] = a.shareApp.List()
	}

	if request.CanWebhook {
		content["Webhooks"] = a.webhookApp.List()
	}

	return renderer.NewPage("files", http.StatusOK, content), nil
}

func (a App) enrichThumbnail(ctx context.Context, directoryAggregate provider.Aggregate, items []provider.RenderItem, thumbnails map[string]absto.Item) (hasThumbnail bool, hasStory bool, cover cover) {
	for index, item := range items {
		if _, ok := thumbnails[a.thumbnailApp.Path(item.Item)]; !ok {
			continue
		}

		hasThumbnail = true

		if cover.IsZero() || (len(directoryAggregate.Cover) != 0 && cover.Img.Name != directoryAggregate.Cover) {
			cover = newCover(item, thumbnail.SmallSize)
		}

		if !hasStory {
			hasStory = a.thumbnailApp.HasLargeThumbnail(ctx, item.Item)
		}

		items[index].HasThumbnail = true
	}

	return
}

// Download content of a directory into a streamed zip
func (a App) Download(w http.ResponseWriter, r *http.Request, request provider.Request, items []absto.Item) {
	zipWriter := zip.NewWriter(w)
	defer func() {
		if closeErr := zipWriter.Close(); closeErr != nil {
			logger.Error("close zip: %s", closeErr)
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
				err = fmt.Errorf("zip nested folder `%s`: %w", relativeURL, err)
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
		Modified:           item.Date,
		Method:             zip.Deflate,
	}
	header.SetMode(0o600)

	var writer io.Writer
	writer, err = zipWriter.CreateHeader(header)
	if err != nil {
		return fmt.Errorf("create zip header: %w", err)
	}

	var reader io.ReadCloser
	reader, err = a.storageApp.ReadFrom(ctx, item.Pathname)
	if err != nil {
		return fmt.Errorf("read: %w", err)
	}

	defer func() {
		err = provider.HandleClose(reader, err)
	}()

	buffer := provider.BufferPool.Get().(*bytes.Buffer)
	defer provider.BufferPool.Put(buffer)

	_, err = io.CopyBuffer(writer, reader, buffer.Bytes())

	return
}
