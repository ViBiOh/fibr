package crud

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"path"
	"strconv"
	"strings"

	"github.com/ViBiOh/fibr/provider"
	"github.com/ViBiOh/fibr/thumbnail"
	"github.com/ViBiOh/fibr/utils"
	"github.com/ViBiOh/httputils"
)

// CheckAndServeSEO check if filename match SEO and serve it, or not
func (a *App) CheckAndServeSEO(w http.ResponseWriter, r *http.Request) bool {
	if r.URL.Path == `/robots.txt` {
		http.ServeFile(w, r, path.Join(`web/static`, r.URL.Path))
		return true
	} else if r.URL.Path == `/sitemap.xml` {
		a.renderer.Sitemap(w)
		return true
	}

	return false
}

// GetDir render directory web view of given dirPath
func (a *App) GetDir(w http.ResponseWriter, config *provider.RequestConfig, filename string, display string, message *provider.Message) {
	files, err := ioutil.ReadDir(filename)
	if err != nil {
		a.renderer.Error(w, http.StatusInternalServerError, err)
		return
	}

	config.Path = strings.TrimPrefix(filename, config.Root)

	content := map[string]interface{}{
		`Files`: files,
	}

	if config.CanShare {
		content[`Shares`] = a.metadatas
	}

	a.renderer.Directory(w, config, content, display, message)
}

// Get write given path from filesystem
func (a *App) Get(w http.ResponseWriter, r *http.Request, config *provider.RequestConfig, message *provider.Message) {
	filename, info := utils.GetPathInfo(config.Root, config.Path)

	if info == nil {
		if !a.CheckAndServeSEO(w, r) {
			a.renderer.Error(w, http.StatusNotFound, fmt.Errorf(`Requested path does not exist: %s`, config.Path))
		}
	} else if info.IsDir() {
		a.GetDir(w, config, filename, r.URL.Query().Get(`d`), message)
	} else {
		values := r.URL.Query()
		size := values.Get(`size`)

		if !provider.ImageExtensions[path.Ext(info.Name())] || size == `` {
			http.ServeFile(w, r, filename)
			return
		}

		parts := strings.Split(size, `x`)
		if len(parts) != 2 {
			httputils.BadRequest(w, errors.New(`Invalid size format, expected 'size=[width]x[height]`))
			return
		}

		width, err := strconv.Atoi(parts[0])
		if err != nil {
			httputils.BadRequest(w, fmt.Errorf(`Invalid width for image size: %v`, err))
			return
		}

		height, err := strconv.Atoi(parts[1])
		if err != nil {
			httputils.BadRequest(w, fmt.Errorf(`Invalid height for image size: %v`, err))
			return
		}

		if err := thumbnail.ServeThumbnail(w, filename, width, height); err != nil {
			httputils.InternalServerError(w, fmt.Errorf(`Error while serving thumbnail: %v`, err))
		}
	}
}
