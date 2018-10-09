package thumbnail

import (
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"path"
	"path/filepath"
	"strings"

	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/httputils/pkg/httperror"
	"github.com/ViBiOh/httputils/pkg/httpjson"
	"github.com/ViBiOh/httputils/pkg/rollbar"
	"github.com/ViBiOh/httputils/pkg/tools"
	"github.com/disintegration/imaging"
)

var (
	ignoredThumbnailDir = map[string]bool{
		`vendor`:       true,
		`vendors`:      true,
		`node_modules`: true,
	}
)

// App stores informations
type App struct {
	storage provider.Storage
}

// NewApp creates new App from Flags' config
func NewApp(storage provider.Storage) *App {
	return &App{
		storage: storage,
	}
}

func getThumbnailPath(pathname string) string {
	return path.Join(provider.MetadataDirectoryName, pathname)
}

// IsExist determine if thumbnail exist for given pathname
func (a App) IsExist(pathname string) bool {
	_, err := a.storage.Info(getThumbnailPath(pathname))
	return err == nil
}

// ServeIfPresent check if thumbnail is present and serve it
func (a App) ServeIfPresent(w http.ResponseWriter, r *http.Request, pathname string) bool {
	exist := a.IsExist(pathname)
	if exist {
		a.storage.Serve(w, r, getThumbnailPath(pathname))
	}

	return exist
}

// List return all thumbnail in a base64 form
func (a App) List(w http.ResponseWriter, r *http.Request, pathname string) {
	items, err := a.storage.List(pathname)
	if err != nil {
		httperror.InternalServerError(w, err)
		return
	}

	thumbnails := make(map[string]string)

	for _, item := range items {
		if item.IsDir || !a.IsExist(item.Pathname) {
			continue
		}

		file, err := a.storage.Read(getThumbnailPath(item.Pathname))
		if err != nil {
			httperror.InternalServerError(w, fmt.Errorf(`error while reading %s: %v`, item.Pathname, err))
			return
		}

		content, err := ioutil.ReadAll(file)
		if err != nil {
			httperror.InternalServerError(w, fmt.Errorf(`error while reading content %s: %v`, item.Pathname, err))
			return
		}

		thumbnails[tools.Sha1(item.Name)] = base64.StdEncoding.EncodeToString(content)
	}

	if err := httpjson.ResponseJSON(w, http.StatusOK, thumbnails, httpjson.IsPretty(r)); err != nil {
		httperror.InternalServerError(w, err)
	}
}

// Generate thumbnail for all storage
func (a App) Generate() {
	err := a.storage.Walk(func(pathname string, item *provider.StorageItem, _ error) error {
		if item.IsDir && strings.HasPrefix(item.Name, `.`) || ignoredThumbnailDir[item.Name] {
			return filepath.SkipDir
		}

		if provider.ImageExtensions[item.Extension()] {
			info, err := a.storage.Info(getThumbnailPath(pathname))
			if err != nil && !provider.IsNotExist(err) {
				return err
			}

			if info == nil {
				a.GenerateImageThumbnail(pathname)
			}
		}

		return nil
	})

	if err != nil {
		rollbar.LogError(`error while walking: %v`, err)
	}
}

// GenerateImageThumbnail generate thumbnail image for given path
func (a App) GenerateImageThumbnail(pathname string) {
	info, err := a.storage.Info(pathname)
	if err != nil && !provider.IsNotExist(err) {
		rollbar.LogError(`error while getting info about %s: %v`, pathname, err)
		return
	}

	if info == nil {
		rollbar.LogError(`emage not found for %s`, pathname)
		return
	}

	thumbnailPath := getThumbnailPath(pathname)

	thumbInfo, err := a.storage.Info(thumbnailPath)
	if err != nil && !provider.IsNotExist(err) {
		rollbar.LogError(`error while getting info about thumbnail for %s: %v`, pathname, err)
		return
	}

	if thumbInfo != nil {
		rollbar.LogError(`ehumbnail already exists for %s`, pathname)
		return
	}

	file, err := a.storage.Read(pathname)
	if file != nil {
		defer func() {
			if err := file.Close(); err != nil {
				rollbar.LogError(`error while closing file %s: %v`, pathname, err)
			}
		}()
	}
	if err != nil {
		rollbar.LogError(`error while opening file %s: %v`, pathname, err)
		return
	}

	sourceImage, err := imaging.Decode(file)
	if err != nil {
		rollbar.LogError(`error while opening file %s: %v`, pathname, err)
		return
	}
	resizedImage := imaging.Fill(sourceImage, 150, 150, imaging.Center, imaging.Box)

	if err := a.storage.Create(path.Dir(thumbnailPath)); err != nil {
		rollbar.LogError(`error while getting creating thumbnail dir for %s: %v`, pathname, err)
		return
	}

	thumbnailFile, err := a.storage.Open(thumbnailPath)
	if thumbnailFile != nil {
		defer func() {
			if err := thumbnailFile.Close(); err != nil {
				rollbar.LogError(`error while closing file %s: %v`, thumbnailPath, err)
			}
		}()
	}
	if err != nil {
		rollbar.LogError(`error while opening thumbnail file %s: %v`, pathname, err)
		return
	}

	format, err := imaging.FormatFromFilename(thumbnailPath)
	if err != nil {
		rollbar.LogError(`error while getting thumbnail format for %s: %v`, pathname, err)
		return
	}

	if err = imaging.Encode(thumbnailFile, resizedImage, format); err != nil {
		rollbar.LogError(`error while saving file for %s: %v`, pathname, err)
		return
	}

	log.Printf(`Generation success for %s`, pathname)
}
