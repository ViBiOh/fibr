package thumbnail

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"time"

	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/httputils/v4/pkg/request"
)

const (
	defaultTimeout = time.Minute * 2
)

var thumbnailClient = &http.Client{
	Timeout: 2 * time.Minute,
	CheckRedirect: func(*http.Request, []*http.Request) error {
		return http.ErrUseLastResponse
	},
}

func (a App) generate(item provider.StorageItem) error {
	var (
		file io.ReadCloser
		err  error
	)

	file, err = a.storageApp.ReaderFrom(item.Pathname) // file will be closed by `.Send`
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	req := request.New().WithClient(thumbnailClient)
	var resp *http.Response

	if item.IsVideo() {
		info, err := a.storageApp.Info(item.Pathname)
		if err != nil {
			return fmt.Errorf("unable to get video info: %s", err)
		}

		r, err := req.Post(fmt.Sprintf("%s/", a.videoURL)).Build(ctx, file)
		if err != nil {
			return fmt.Errorf("unable to create video request: %s", err)
		}

		r.ContentLength = info.Size

		resp, err = request.DoWithClient(thumbnailClient, r)
		if err != nil {
			return err
		}

		file = resp.Body
	}

	a.increaseMetric("requested")

	resp, err = req.Post(a.imageURL).Send(ctx, file)
	if err != nil {
		return err
	}

	thumbnailPath := getThumbnailPath(item)
	if err := a.storageApp.CreateDir(filepath.Dir(thumbnailPath)); err != nil {
		return err
	}

	if err := a.storageApp.Store(thumbnailPath, resp.Body); err != nil {
		return err
	}

	a.increaseMetric("saved")

	return nil
}
