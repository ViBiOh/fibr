package crud

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"path"
	"syscall"

	absto "github.com/ViBiOh/absto/pkg/model"
	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/fibr/pkg/search"
	"github.com/ViBiOh/fibr/pkg/thumbnail"
	"github.com/ViBiOh/httputils/v4/pkg/concurrent"
	"github.com/ViBiOh/httputils/v4/pkg/renderer"
	"github.com/ViBiOh/httputils/v4/pkg/telemetry"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

func listLogger(pathname string) *slog.Logger {
	return slog.With("fn", "crud.list").With("item", pathname)
}

func (s Service) list(ctx context.Context, request provider.Request, message renderer.Message, item absto.Item, files []absto.Item) (renderer.Page, error) {
	ctx, end := telemetry.StartSpan(ctx, s.tracer, "list", trace.WithAttributes(attribute.String("item", item.Pathname)))
	defer end(nil)

	wg := concurrent.NewLimiter(-1)

	var directoryAggregate provider.Aggregate
	wg.Go(func() {
		var err error

		directoryAggregate, err = s.metadata.GetAggregateFor(ctx, item)
		if err != nil && !absto.IsNotExist(err) {
			listLogger(item.Pathname).ErrorContext(ctx, "get aggregate", "error", err)
		}
	})

	var aggregates map[string]provider.Aggregate
	wg.Go(func() {
		var err error

		aggregates, err = s.metadata.GetAllAggregateFor(ctx, files...)
		if err != nil {
			listLogger(item.Pathname).ErrorContext(ctx, "list aggregates", "error", err)
		}
	})

	var metadatas map[string]provider.Metadata
	wg.Go(func() {
		var err error

		metadatas, err = s.metadata.GetAllMetadataFor(ctx, files...)
		if err != nil {
			listLogger(item.Pathname).ErrorContext(ctx, "list metadatas", "error", err)
		}
	})

	var thumbnails map[string]absto.Item
	thumbnailDone := make(chan struct{})
	go func() {
		defer close(thumbnailDone)

		var err error

		thumbnails, err = s.thumbnail.ListDir(ctx, item)
		if err != nil {
			listLogger(item.Pathname).ErrorContext(ctx, "list thumbnail", "error", err)
			return
		}
	}()

	var savedSearches search.Searches
	savedSearchDone := make(chan struct{})
	go func() {
		defer close(savedSearchDone)

		var err error

		savedSearches, err = s.searchService.List(ctx, item)
		if err != nil {
			listLogger(item.Pathname).ErrorContext(ctx, "list saved searches", "error", err)
			return
		}
	}()

	wg.Wait()

	items := make([]provider.RenderItem, len(files))

	for index, item := range files {
		renderItem := provider.StorageToRender(item, request)
		renderItem.Tags = metadatas[item.ID].Tags

		if item.IsDir() {
			renderItem.Aggregate = aggregates[item.ID]
		} else {
			renderItem.IsCover = item.Name() == directoryAggregate.Cover
		}

		items[index] = renderItem
	}

	<-thumbnailDone
	hasThumbnail, hasStory, cover := s.enrichThumbnail(ctx, directoryAggregate, items, thumbnails)

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
		"ThumbnailSize": thumbnail.SmallSize,
		"ChunkUpload":   s.chunkUpload,
	}

	if request.CanShare {
		content["Shares"] = s.share.List()
	}

	if request.CanWebhook {
		content["Webhooks"] = s.webhook.List()
	}

	return renderer.NewPage("files", http.StatusOK, content), nil
}

func (s Service) enrichThumbnail(ctx context.Context, directoryAggregate provider.Aggregate, items []provider.RenderItem, thumbnails map[string]absto.Item) (hasThumbnail bool, hasStory bool, cover cover) {
	for index, item := range items {
		if _, ok := thumbnails[s.thumbnail.Path(item.Item)]; !ok {
			continue
		}

		hasThumbnail = true

		if cover.IsZero() || (len(directoryAggregate.Cover) != 0 && cover.Img.Name() != directoryAggregate.Cover) {
			cover = newCover(item, thumbnail.SmallSize)
		}

		if !hasStory {
			hasStory = s.thumbnail.HasLargeThumbnail(ctx, item.Item)
		}

		items[index].HasThumbnail = true
	}

	return
}

func (s Service) Download(w http.ResponseWriter, r *http.Request, request provider.Request, items []absto.Item) {
	zipWriter := zip.NewWriter(w)

	ctx := r.Context()

	defer func() {
		if closeErr := zipWriter.Close(); closeErr != nil {
			slog.LogAttrs(ctx, slog.LevelError, "close zip", slog.Any("error", closeErr))
		}
	}()

	filename := path.Base(request.Path)
	if filename == "/" && !request.Share.IsZero() {
		filename = path.Base(path.Join(request.Share.RootName, request.Path))
	}

	w.Header().Add("Content-Disposition", fmt.Sprintf("attachment; filename=%s.zip", filename))

	if err := s.zipItems(ctx, ctx.Done(), request, zipWriter, items); err != nil {
		s.error(w, r, request, err)
	}
}

func (s Service) zipItems(ctx context.Context, done <-chan struct{}, request provider.Request, zipWriter *zip.Writer, items []absto.Item) (err error) {
	defer func() {
		if errors.Is(err, syscall.ECONNRESET) {
			err = nil
		}
	}()

	for _, item := range items {
		select {
		case <-done:
			slog.ErrorContext(ctx, "context is done for zipping files")
			return nil

		default:
			relativeURL := request.RelativeURL(item)

			if !item.IsDir() {
				if err = s.addFileToZip(ctx, zipWriter, item, relativeURL); err != nil {
					return
				}

				continue
			}

			var nestedItems []absto.Item
			nestedItems, err = s.storage.List(ctx, request.SubPath(relativeURL))
			if err != nil {
				err = fmt.Errorf("zip nested folder `%s`: %w", relativeURL, err)
				return
			}

			if err = s.zipItems(ctx, done, request, zipWriter, nestedItems); err != nil {
				return err
			}
		}
	}

	return nil
}

func (s Service) addFileToZip(ctx context.Context, zipWriter *zip.Writer, item absto.Item, pathname string) (err error) {
	header := &zip.FileHeader{
		Name:               pathname,
		UncompressedSize64: uint64(item.Size()),
		Modified:           item.Date,
		Method:             zip.Deflate,
	}
	header.SetMode(absto.RegularFilePerm)

	var writer io.Writer
	writer, err = zipWriter.CreateHeader(header)
	if err != nil {
		return fmt.Errorf("create zip header: %w", err)
	}

	var reader io.ReadCloser
	reader, err = s.storage.ReadFrom(ctx, item.Pathname)
	if err != nil {
		return fmt.Errorf("read: %w", err)
	}

	defer func() {
		err = errors.Join(err, reader.Close())
	}()

	buffer := provider.BufferPool.Get().(*bytes.Buffer)
	defer provider.BufferPool.Put(buffer)

	_, err = io.CopyBuffer(writer, reader, buffer.Bytes())

	return
}
