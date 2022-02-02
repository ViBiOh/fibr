package crud

import (
	"encoding/json"
	"fmt"
	"net/http"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"

	absto "github.com/ViBiOh/absto/pkg/model"
	"github.com/ViBiOh/fibr/pkg/geo"
	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/httputils/v4/pkg/logger"
	"github.com/ViBiOh/httputils/v4/pkg/model"
	"github.com/ViBiOh/httputils/v4/pkg/query"
	"github.com/ViBiOh/httputils/v4/pkg/renderer"
)

func (a App) getWithMessage(w http.ResponseWriter, r *http.Request, request provider.Request, message renderer.Message) (string, int, map[string]interface{}, error) {
	pathname := request.Filepath()
	item, err := a.storageApp.Info(pathname)

	if err != nil && absto.IsNotExist(err) && provider.StreamExtensions[filepath.Ext(pathname)] {
		logger.Info("share=`%+v`, pathname=`%s`, dir=`%s`, dir of dir=`%s`", request.Share, pathname, path.Dir(pathname), path.Dir(path.Dir(pathname)))
		if request.Share.File {
			// URL with /<share_id>/segment.ts witll be converted to `/path/of/shared/file/segment.ts`, so we need to remove two directories before appending segment
			pathname = path.Dir(path.Dir(pathname)) + "/" + path.Base(pathname)
		}

		item, err = a.thumbnailApp.GetChunk(pathname)
	}

	if err != nil {
		if absto.IsNotExist(err) {
			return errorReturn(request, model.WrapNotFound(err))
		}

		return errorReturn(request, model.WrapNotFound(err))
	}

	if item.IsDir && !strings.HasSuffix(r.URL.Path, "/") {
		a.rendererApp.Redirect(w, r, fmt.Sprintf("%s/?d=%s", r.URL.Path, request.Display), renderer.Message{})
		return "", 0, nil, nil
	}

	if !item.IsDir {
		return a.handleFile(w, r, request, item, message)
	}
	return a.handleDir(w, r, request, item, message)
}

func (a App) handleFile(w http.ResponseWriter, r *http.Request, request provider.Request, item absto.Item, message renderer.Message) (string, int, map[string]interface{}, error) {
	if query.GetBool(r, "thumbnail") {
		a.thumbnailApp.Serve(w, r, item)
		return "", 0, nil, nil
	}

	if query.GetBool(r, "stream") {
		a.thumbnailApp.Stream(w, r, item)
		return "", 0, nil, nil
	}

	if query.GetBool(r, "browser") {
		provider.SetPrefsCookie(w, request)

		go a.notify(provider.NewAccessEvent(item, r))

		return a.Browser(w, request, item, message)
	}

	return "", 0, nil, a.serveFile(w, r, item)
}

func (a App) serveFile(w http.ResponseWriter, r *http.Request, item absto.Item) error {
	file, err := a.storageApp.ReadFrom(item.Pathname)
	if err != nil {
		return fmt.Errorf("unable to get reader for `%s`: %w", item.Pathname, err)
	}

	defer func() {
		if err = file.Close(); err != nil {
			logger.WithField("fn", "crud.serveFile").WithField("item", item.Pathname).Error("unable to close: %s", err)
		}
	}()

	http.ServeContent(w, r, item.Name, item.Date, file)
	return nil
}

func (a App) handleDir(w http.ResponseWriter, r *http.Request, request provider.Request, item absto.Item, message renderer.Message) (string, int, map[string]interface{}, error) {
	if query.GetBool(r, "stats") {
		return a.Stats(w, request, message)
	}

	items, err := a.listFiles(r, request)
	if err != nil {
		return errorReturn(request, err)
	}

	if query.GetBool(r, "geojson") {
		a.serveGeoJSON(w, r, request, items)
		return "", 0, nil, nil
	}

	if query.GetBool(r, "thumbnail") {
		a.thumbnailApp.List(w, r, items)
		return "", 0, nil, nil
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
	return a.List(request, message, item, items)
}

func (a App) listFiles(r *http.Request, request provider.Request) (items []absto.Item, err error) {
	if query.GetBool(r, "search") {
		items, err = a.searchFiles(r, request)
	} else {
		items, err = a.storageApp.List(request.Filepath())
	}

	sort.Sort(provider.ByHybridSort(items))

	return items, err
}

func (a App) serveGeoJSON(w http.ResponseWriter, r *http.Request, request provider.Request, items []absto.Item) {
	if len(items) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	w.Header().Add("Content-Type", "application/json; charset=utf-8")
	w.Header().Add("Cache-Control", "no-cache")
	w.WriteHeader(http.StatusOK)

	done := r.Context().Done()
	isDone := func() bool {
		select {
		case <-done:
			return true
		default:
			return false
		}
	}

	var commaNeeded bool
	encoder := json.NewEncoder(w)

	provider.SafeWrite(w, `{"type":"FeatureCollection","features":[`)

	for _, item := range items {
		if isDone() {
			return
		}

		exif, err := a.exifApp.GetExifFor(item)
		if err != nil {
			logger.WithField("item", item.Pathname).Error("unable to get exif: %s", err)
		}

		if exif.Geocode.Longitude == 0 && exif.Geocode.Latitude == 0 {
			continue
		}

		if commaNeeded {
			provider.DoneWriter(isDone, w, ",")
		}

		point := geo.NewPoint(geo.NewPosition(exif.Geocode.Longitude, exif.Geocode.Latitude))
		if err := encoder.Encode(geo.NewFeature(&point, map[string]interface{}{
			"url":  request.RelativeURL(item),
			"date": exif.Date.Format(time.RFC850),
		})); err != nil {
			logger.WithField("item", item.Pathname).Error("unable to encode feature: %s", err)
		}

		commaNeeded = true
	}

	provider.SafeWrite(w, "]}")
}

// Get output content
func (a App) Get(w http.ResponseWriter, r *http.Request, request provider.Request) (string, int, map[string]interface{}, error) {
	return a.getWithMessage(w, r, request, renderer.ParseMessage(r))
}
