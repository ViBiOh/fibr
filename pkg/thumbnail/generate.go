package thumbnail

import (
	"context"
	"fmt"
	"net/http"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/httputils/v3/pkg/logger"
	"github.com/ViBiOh/httputils/v3/pkg/request"
)

const (
	waitTimeout = time.Millisecond * 300
)

func (a app) generateDir(pathname string) error {
	return a.storage.Walk(pathname, func(item provider.StorageItem, _ error) error {
		if item.IsDir && strings.HasPrefix(item.Name, ".") || ignoredThumbnailDir[item.Name] {
			return filepath.SkipDir
		}

		if !CanHaveThumbnail(item) || a.HasThumbnail(item) {
			return nil
		}

		a.AsyncGenerateThumbnail(item)

		return nil
	})
}

func (a app) generateThumbnail(item provider.StorageItem) error {
	file, err := a.storage.ReaderFrom(item.Pathname)
	if err != nil {
		return err
	}

	ctx, cancel := getCtx(context.Background())
	defer cancel()

	var resp *http.Response

	if item.IsVideo() {
		resp, err = request.New().Post(fmt.Sprintf("%s/", a.vithURL)).Send(ctx, file)
		if err != nil {
			return err
		}

		file = resp.Body
	}

	resp, err = request.New().Post(a.imaginaryURL).Send(ctx, file)
	if err != nil {
		return err
	}

	thumbnailPath := getThumbnailPath(item)
	if err := a.storage.CreateDir(path.Dir(thumbnailPath)); err != nil {
		return err
	}

	if err := a.storage.Store(thumbnailPath, resp.Body); err != nil {
		return err
	}

	return nil
}

// Generate thumbnail for all storage
func (a app) Generate() {
	if !a.Enabled() {
		return
	}

	if err := a.generateDir(""); err != nil {
		logger.Error("error while walking root dir: %s", err)
	}
}

// AsyncGenerateThumbnail generate thumbnail image for given path
func (a app) AsyncGenerateThumbnail(item provider.StorageItem) {
	if !a.Enabled() {
		return
	}

	a.pathnameInput <- item
}

func (a app) Start() {
	if !a.Enabled() {
		return
	}

	for item := range a.pathnameInput {
		if err := a.generateThumbnail(item); err != nil {
			logger.Error("unable to generate thumbnail for %s: %s", item.Pathname, err)
		} else {
			logger.Info("Thumbnail generated for %s", item.Pathname)
		}

		// Do not stress API
		time.Sleep(waitTimeout)
	}
}
