package crud

import (
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/ViBiOh/fibr/pkg/geo"
	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/httputils/v4/pkg/httperror"
	"github.com/ViBiOh/httputils/v4/pkg/httpjson"
	"github.com/ViBiOh/httputils/v4/pkg/logger"
	"github.com/ViBiOh/httputils/v4/pkg/model"
	"github.com/ViBiOh/httputils/v4/pkg/query"
	"github.com/ViBiOh/httputils/v4/pkg/renderer"
)

func (a App) getWithMessage(w http.ResponseWriter, r *http.Request, request provider.Request, message renderer.Message) (string, int, map[string]interface{}, error) {
	if query.GetBool(r, "stats") {
		return a.Stats(w, request, message)
	}

	pathname := request.GetFilepath("")
	item, err := a.storageApp.Info(pathname)

	if err != nil && provider.IsNotExist(err) && provider.StreamExtensions[filepath.Ext(pathname)] {
		item, err = a.thumbnailApp.GetChunk(pathname)
	}

	if err != nil {
		if provider.IsNotExist(err) {
			return "", 0, nil, model.WrapNotFound(err)
		}

		return "", 0, nil, model.WrapInternal(err)
	}

	if query.GetBool(r, "geojson") {
		a.serveGeoJSON(w, r, request, item)
		return "", 0, nil, nil
	}

	if query.GetBool(r, "thumbnail") {
		a.serveThumbnail(w, r, item)
		return "", 0, nil, nil
	}

	if query.GetBool(r, "stream") {
		a.thumbnailApp.Stream(w, r, item)
		return "", 0, nil, nil
	}

	if item.IsDir && !strings.HasSuffix(r.URL.Path, "/") {
		a.rendererApp.Redirect(w, r, fmt.Sprintf("%s/?d=%s", r.URL.Path, request.Layout("")), renderer.Message{})
		return "", 0, nil, nil
	}

	go a.notify(provider.NewAccessEvent(item, r))

	if !item.IsDir {
		if query.GetBool(r, "browser") {
			provider.SetPrefsCookie(w, request)
			return a.Browser(w, request, item, message)
		}

		return "", 0, nil, a.serveFile(w, r, item)
	}

	if query.GetBool(r, "download") {
		a.Download(w, r, request)
		return "", 0, nil, err
	}

	provider.SetPrefsCookie(w, request)
	return a.List(w, request, message)
}

func (a App) serveGeoJSON(w http.ResponseWriter, r *http.Request, request provider.Request, item provider.StorageItem) {
	if !item.IsDir {
		w.WriteHeader(http.StatusNoContent)
	}

	items, err := a.storageApp.List(item.Pathname)
	if err != nil {
		httperror.InternalServerError(w, err)
		return
	}

	var features []geo.Feature

	for _, item := range items {
		exif, err := a.exifApp.GetExifFor(item)
		if err != nil {
			logger.WithField("item", item.Pathname).Error("unable to get exif: %s", err)
		}

		if exif.Geocode.Longitude != 0 && exif.Geocode.Latitude != 0 {
			point := geo.NewPoint(geo.NewPosition(exif.Geocode.Longitude, exif.Geocode.Latitude, 0))
			features = append(features, geo.NewFeature(&point, map[string]interface{}{
				"name": item.Name,
				"date": exif.Date.Format(time.RFC850),
			}))
		}
	}

	httpjson.Write(w, http.StatusOK, geo.NewFeatureCollection(features))
}

func (a App) serveThumbnail(w http.ResponseWriter, r *http.Request, item provider.StorageItem) {
	if item.IsDir {
		w.WriteHeader(http.StatusNotFound)
	} else {
		a.thumbnailApp.Serve(w, r, item)
	}
}

func (a App) serveFile(w http.ResponseWriter, r *http.Request, item provider.StorageItem) error {
	file, err := a.storageApp.ReaderFrom(item.Pathname)
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

// Get output content
func (a App) Get(w http.ResponseWriter, r *http.Request, request provider.Request) (string, int, map[string]interface{}, error) {
	return a.getWithMessage(w, r, request, renderer.ParseMessage(r))
}
