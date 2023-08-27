package crud

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"

	absto "github.com/ViBiOh/absto/pkg/model"
	"github.com/ViBiOh/fibr/pkg/geo"
	"github.com/ViBiOh/fibr/pkg/metadata"
	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/httputils/v4/pkg/cntxt"
	"github.com/ViBiOh/httputils/v4/pkg/hash"
	"github.com/ViBiOh/httputils/v4/pkg/model"
	"github.com/ViBiOh/httputils/v4/pkg/query"
	"github.com/ViBiOh/httputils/v4/pkg/renderer"
	"github.com/ViBiOh/httputils/v4/pkg/telemetry"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

func (s Service) getWithMessage(w http.ResponseWriter, r *http.Request, request provider.Request, message renderer.Message) (renderer.Page, error) {
	ctx := r.Context()

	pathname := request.Filepath()
	item, err := s.storage.Stat(ctx, pathname)

	if err != nil && absto.IsNotExist(err) && provider.StreamExtensions[filepath.Ext(pathname)] {
		if request.Share.File {
			// URL with /<share_id>/segment.ts will be the path `/path/of/shared/file/segment.ts`, so we need to remove two directories before appending segment
			pathname = provider.Dirname(path.Dir(path.Dir(pathname))) + path.Base(pathname)
		}

		item, err = s.thumbnail.GetChunk(ctx, pathname)
	}

	if err != nil {
		if absto.IsNotExist(err) {
			err = model.WrapNotFound(err)
		}

		return errorReturn(request, err)
	}

	if item.IsDir() && !strings.HasSuffix(r.URL.Path, "/") {
		s.renderer.Redirect(w, r, fmt.Sprintf("%s/?d=%s", r.URL.Path, request.Display), renderer.Message{})
		return renderer.Page{}, nil
	}

	if !item.IsDir() {
		return s.handleFile(w, r, request, item, message)
	}
	return s.handleDir(w, r, request, item, message)
}

func (s Service) handleFile(w http.ResponseWriter, r *http.Request, request provider.Request, item absto.Item, message renderer.Message) (renderer.Page, error) {
	if query.GetBool(r, "thumbnail") {
		s.thumbnail.Serve(w, r, item)
		return renderer.Page{}, nil
	}

	if query.GetBool(r, "stream") {
		s.thumbnail.Stream(w, r, item)
		return renderer.Page{}, nil
	}

	if query.GetBool(r, "browser") {
		provider.SetPrefsCookie(w, request)

		go s.pushEvent(cntxt.WithoutDeadline(r.Context()), provider.NewAccessEvent(r.Context(), item, r))

		return s.browse(r.Context(), request, item, message)
	}

	return renderer.Page{}, s.serveFile(w, r, item)
}

func (s Service) serveFile(w http.ResponseWriter, r *http.Request, item absto.Item) error {
	var err error

	ctx, end := telemetry.StartSpan(r.Context(), s.tracer, "file", trace.WithSpanKind(trace.SpanKindInternal))
	defer end(&err)

	etag, ok := provider.EtagMatch(w, r, provider.Hash(item.String()))
	if ok {
		return nil
	}

	file, err := s.storage.ReadFrom(ctx, item.Pathname)
	if err != nil {
		return fmt.Errorf("get reader for `%s`: %w", item.Pathname, err)
	}

	defer provider.LogClose(file, "crud.serveFile", item.Pathname)

	w.Header().Add("Etag", etag)

	http.ServeContent(w, r, item.Name(), item.Date, file)
	return nil
}

func (s Service) handleDir(w http.ResponseWriter, r *http.Request, request provider.Request, item absto.Item, message renderer.Message) (renderer.Page, error) {
	if query.GetBool(r, "stats") {
		return s.stats(r, request, message)
	}

	items, err := s.listFiles(r, request, item)
	if err != nil {
		return errorReturn(request, err)
	}

	if query.GetBool(r, "geojson") {
		s.serveGeoJSON(w, r, request, item, items)
		return renderer.Page{}, nil
	}

	if query.GetBool(r, "thumbnail") {
		s.thumbnail.List(w, r, item, items)
		return renderer.Page{}, nil
	}

	if query.GetBool(r, "download") {
		s.Download(w, r, request, items)
		return errorReturn(request, err)
	}

	go s.pushEvent(cntxt.WithoutDeadline(r.Context()), provider.NewAccessEvent(r.Context(), item, r))

	if query.GetBool(r, "search") {
		return s.search(r, request, item, items)
	}

	provider.SetPrefsCookie(w, request)

	if request.IsStory() {
		return s.story(r, request, item, items)
	}

	return s.list(r.Context(), request, message, item, items)
}

func (s Service) listFiles(r *http.Request, request provider.Request, item absto.Item) (items []absto.Item, err error) {
	ctx, end := telemetry.StartSpan(r.Context(), s.tracer, "files", trace.WithAttributes(attribute.String("item", item.Pathname)))
	defer end(&err)

	if query.GetBool(r, "search") {
		items, err = s.searchService.Files(r, request)
	} else {
		items, err = s.storage.List(ctx, request.Filepath())
	}

	if request.IsStory() {
		thumbnails, err := s.thumbnail.ListDirLarge(ctx, item)
		if err != nil {
			slog.Error("list large thumbnails", "err", err, "item", item.Pathname)
		}

		storyItems := items[:0]
		for _, item := range items {
			if _, ok := thumbnails[s.thumbnail.PathForLarge(item)]; ok {
				storyItems = append(storyItems, item)
			}
		}
		items = storyItems
	}

	sort.Sort(provider.ByHybridSort(items))

	return items, err
}

func (s Service) serveGeoJSON(w http.ResponseWriter, r *http.Request, request provider.Request, item absto.Item, items []absto.Item) {
	if len(items) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	ctx, end := telemetry.StartSpan(r.Context(), s.tracer, "geojson", trace.WithSpanKind(trace.SpanKindInternal))
	defer end(nil)

	var hash string
	if query.GetBool(r, "search") {
		hash = s.exifHash(ctx, items)
	} else if exifs, err := s.metadata.ListDir(ctx, item); err != nil {
		slog.Error("list exifs", "err", err, "item", item.Pathname)
	} else {
		hash = provider.RawHash(exifs)
	}

	etag, ok := provider.EtagMatch(w, r, hash)
	if ok {
		return
	}

	exifs, err := s.metadata.GetAllMetadataFor(ctx, items...)
	if err != nil {
		s.error(w, r, request, err)
	}

	w.Header().Add("Content-Type", "application/json; charset=utf-8")
	w.Header().Add("Cache-Control", "no-cache")
	w.Header().Add("Etag", etag)
	w.WriteHeader(http.StatusOK)

	s.generateGeoJSON(ctx, w, request, items, exifs)
}

func (s Service) generateGeoJSON(ctx context.Context, w io.Writer, request provider.Request, items []absto.Item, exifs map[string]provider.Metadata) {
	done := ctx.Done()
	isDone := func() bool {
		select {
		case <-done:
			return true
		default:
			return false
		}
	}

	sort.Sort(provider.ByID(items))

	var commaNeeded bool
	encoder := json.NewEncoder(w)

	provider.SafeWrite(w, `{"type":"FeatureCollection","features":[`)

	point := geo.NewPoint(geo.NewPosition(0, 0))
	feature := geo.NewFeature(&point, map[string]any{})

	for id, exif := range exifs {
		if isDone() {
			return
		}

		if !exif.Geocode.HasCoordinates() {
			continue
		}

		item := dichotomicFind(items, id)
		if item.IsZero() {
			continue
		}

		if commaNeeded {
			provider.DoneWriter(isDone, w, ",")
		} else {
			commaNeeded = true
		}

		point.Coordinates.Latitude = exif.Geocode.Latitude
		point.Coordinates.Longitude = exif.Geocode.Longitude

		feature.Properties["url"] = request.RelativeURL(item)
		feature.Properties["date"] = exif.Date.Format(time.RFC850)

		if err := encoder.Encode(feature); err != nil {
			slog.Error("encode feature", "err", err, "item", item.Pathname)
		}
	}

	provider.SafeWrite(w, "]}")
}

func dichotomicFind(items []absto.Item, id string) absto.Item {
	min := 0
	max := len(items) - 1

	for min <= max {
		current := (min + max) / 2

		item := items[current]
		if item.ID == id {
			return item
		}

		if id < item.ID {
			max = current - 1
		} else {
			min = current + 1
		}
	}

	return absto.Item{}
}

func (s Service) exifHash(ctx context.Context, items []absto.Item) string {
	hasher := hash.Stream()

	for _, item := range items {
		if info, err := s.storage.Stat(ctx, metadata.Path(item)); err == nil {
			hasher.Write(info)
		}
	}

	return hasher.Sum()
}

func (s Service) Get(w http.ResponseWriter, r *http.Request, request provider.Request) (renderer.Page, error) {
	return s.getWithMessage(w, r, request, renderer.ParseMessage(r))
}
