package crud

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"

	absto "github.com/ViBiOh/absto/pkg/model"
	exas "github.com/ViBiOh/exas/pkg/model"
	"github.com/ViBiOh/fibr/pkg/exif"
	"github.com/ViBiOh/fibr/pkg/geo"
	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/httputils/v4/pkg/logger"
	"github.com/ViBiOh/httputils/v4/pkg/model"
	"github.com/ViBiOh/httputils/v4/pkg/query"
	"github.com/ViBiOh/httputils/v4/pkg/renderer"
	"github.com/ViBiOh/httputils/v4/pkg/sha"
	"github.com/ViBiOh/httputils/v4/pkg/tracer"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

func (a App) getWithMessage(w http.ResponseWriter, r *http.Request, request provider.Request, message renderer.Message) (renderer.Page, error) {
	ctx := r.Context()

	pathname := request.Filepath()
	item, err := a.storageApp.Info(ctx, pathname)

	if err != nil && absto.IsNotExist(err) && provider.StreamExtensions[filepath.Ext(pathname)] {
		if request.Share.File {
			// URL with /<share_id>/segment.ts will be the pas `/path/of/shared/file/segment.ts`, so we need to remove two directories before appending segment
			pathname = provider.Dirname(path.Dir(path.Dir(pathname))) + path.Base(pathname)
		}

		item, err = a.thumbnailApp.GetChunk(ctx, pathname)
	}

	if err != nil {
		if absto.IsNotExist(err) {
			return errorReturn(request, model.WrapNotFound(err))
		}

		return errorReturn(request, model.WrapNotFound(err))
	}

	if item.IsDir && !strings.HasSuffix(r.URL.Path, "/") {
		a.rendererApp.Redirect(w, r, fmt.Sprintf("%s/?d=%s", r.URL.Path, request.Display), renderer.Message{})
		return renderer.Page{}, nil
	}

	if !item.IsDir {
		return a.handleFile(w, r, request, item, message)
	}
	return a.handleDir(w, r, request, item, message)
}

func (a App) handleFile(w http.ResponseWriter, r *http.Request, request provider.Request, item absto.Item, message renderer.Message) (renderer.Page, error) {
	if query.GetBool(r, "thumbnail") {
		a.thumbnailApp.Serve(w, r, item)
		return renderer.Page{}, nil
	}

	if query.GetBool(r, "stream") {
		a.thumbnailApp.Stream(w, r, item)
		return renderer.Page{}, nil
	}

	if query.GetBool(r, "browser") {
		provider.SetPrefsCookie(w, request)

		go a.notify(provider.NewAccessEvent(item, r))

		return a.browse(r.Context(), request, item, message)
	}

	return renderer.Page{}, a.serveFile(w, r, item)
}

func (a App) serveFile(w http.ResponseWriter, r *http.Request, item absto.Item) error {
	etag, ok := provider.EtagMatch(w, r, sha.New(item))
	if ok {
		return nil
	}

	file, err := a.storageApp.ReadFrom(r.Context(), item.Pathname)
	if err != nil {
		return fmt.Errorf("get reader for `%s`: %w", item.Pathname, err)
	}

	defer provider.LogClose(file, "crud.serveFile", item.Pathname)

	w.Header().Add("Etag", etag)

	http.ServeContent(w, r, item.Name, item.Date, file)
	return nil
}

func (a App) handleDir(w http.ResponseWriter, r *http.Request, request provider.Request, item absto.Item, message renderer.Message) (renderer.Page, error) {
	if query.GetBool(r, "stats") {
		return a.stats(r, request, message)
	}

	items, err := a.listFiles(r, request, item)
	if err != nil {
		return errorReturn(request, err)
	}

	if query.GetBool(r, "geojson") {
		a.serveGeoJSON(w, r, request, item, items)
		return renderer.Page{}, nil
	}

	if query.GetBool(r, "thumbnail") {
		a.thumbnailApp.List(w, r, item, items)
		return renderer.Page{}, nil
	}

	if query.GetBool(r, "download") {
		a.Download(w, r, request, items)
		return errorReturn(request, err)
	}

	go a.notify(provider.NewAccessEvent(item, r))

	if query.GetBool(r, "search") {
		return a.search(r, request, items)
	}

	provider.SetPrefsCookie(w, request)

	if request.IsStory() {
		return a.story(r, request, item, items)
	}

	return a.list(r.Context(), request, message, item, items)
}

func (a App) listFiles(r *http.Request, request provider.Request, item absto.Item) (items []absto.Item, err error) {
	ctx, end := tracer.StartSpan(r.Context(), a.tracer, "files", trace.WithAttributes(attribute.String("item", item.Pathname)))
	defer end()

	if query.GetBool(r, "search") {
		items, err = a.searchFiles(r, request)
	} else {
		items, err = a.storageApp.List(ctx, request.Filepath())
	}

	if request.IsStory() {
		thumbnails, err := a.thumbnailApp.ListDirLarge(ctx, item)
		if err != nil {
			logger.WithField("item", item.Pathname).Error("list large thumbnails: %s", err)
		}

		storyItems := items[:0]
		for _, item := range items {
			if _, ok := thumbnails[a.thumbnailApp.PathForLarge(item)]; ok {
				storyItems = append(storyItems, item)
			}
		}
		items = storyItems
	}

	sort.Sort(provider.ByHybridSort(items))

	return items, err
}

func (a App) serveGeoJSON(w http.ResponseWriter, r *http.Request, request provider.Request, item absto.Item, items []absto.Item) {
	if len(items) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	ctx, end := tracer.StartSpan(r.Context(), a.tracer, "geojson")
	defer end()

	var hash string
	if query.GetBool(r, "search") {
		hash = a.exifHash(ctx, items)
	} else if exifs, err := a.exifApp.ListDir(ctx, item); err != nil {
		logger.WithField("item", item.Pathname).Error("list exifs: %s", err)
	} else {
		hash = sha.New(exifs)
	}

	etag, ok := provider.EtagMatch(w, r, hash)
	if ok {
		return
	}

	exifs, err := a.exifApp.ListExifFor(ctx, items...)
	if err != nil {
		a.error(w, r, request, err)
	}

	w.Header().Add("Content-Type", "application/json; charset=utf-8")
	w.Header().Add("Cache-Control", "no-cache")
	w.Header().Add("Etag", etag)
	w.WriteHeader(http.StatusOK)

	a.generateGeoJSON(ctx, w, request, items, exifs)
}

func (a App) generateGeoJSON(ctx context.Context, w io.Writer, request provider.Request, items []absto.Item, exifs map[string]exas.Exif) {
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
			logger.WithField("item", item.Pathname).Error("encode feature: %s", err)
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

		if item.ID < id {
			max = current - 1
		} else {
			min = current + 1
		}
	}

	return absto.Item{}
}

func (a App) exifHash(ctx context.Context, items []absto.Item) string {
	hasher := sha.Stream()

	for _, item := range items {
		if info, err := a.storageApp.Info(ctx, exif.Path(item)); err == nil {
			hasher.Write(info)
		}
	}

	return hasher.Sum()
}

func (a App) Get(w http.ResponseWriter, r *http.Request, request provider.Request) (renderer.Page, error) {
	return a.getWithMessage(w, r, request, renderer.ParseMessage(r))
}
