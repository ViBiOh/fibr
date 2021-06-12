package crud

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/httputils/v4/pkg/logger"
	"github.com/ViBiOh/httputils/v4/pkg/model"
	"github.com/ViBiOh/httputils/v4/pkg/query"
	"github.com/ViBiOh/httputils/v4/pkg/renderer"
)

func (a *app) getWithMessage(w http.ResponseWriter, r *http.Request, request provider.Request, message renderer.Message) (string, int, map[string]interface{}, error) {
	info, err := a.storageApp.Info(request.GetFilepath(""))
	if err != nil {
		if provider.IsNotExist(err) {
			return "", 0, nil, model.WrapNotFound(err)
		}

		return "", 0, nil, model.WrapInternal(err)
	}

	if query.GetBool(r, "thumbnail") {
		if info.IsDir {
			a.thumbnailApp.List(w, r, info)
		} else {
			a.thumbnailApp.Serve(w, r, info)
		}

		return "", 0, nil, nil
	}

	if !info.IsDir {
		if query.GetBool(r, "browser") {
			provider.SetPrefsCookie(w, request)
			return a.Browser(w, request, info, message)
		}

		file, err := a.storageApp.ReaderFrom(info.Pathname)
		if file != nil {
			defer func() {
				if err := file.Close(); err != nil {
					logger.Error("unable to close file: %s", err)
				}
			}()
		}
		if err == nil {
			http.ServeContent(w, r, info.Name, info.Date, file)
		}

		return "", 0, nil, err
	}

	if query.GetBool(r, "download") {
		a.Download(w, request)
		return "", 0, nil, err
	}

	if !strings.HasSuffix(r.URL.Path, "/") {
		a.rendererApp.Redirect(w, r, fmt.Sprintf("%s/", r.URL.Path), renderer.Message{})
		return "", 0, nil, nil
	}

	provider.SetPrefsCookie(w, request)
	return a.List(w, request, message)
}

// Get output content
func (a *app) Get(w http.ResponseWriter, r *http.Request, request provider.Request) (string, int, map[string]interface{}, error) {
	return a.getWithMessage(w, r, request, renderer.ParseMessage(r))
}
