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

	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	req := request.New().WithClient(thumbnailClient)
	var resp *http.Response

	if item.IsVideo() {
		resp, err = a.requestVith(ctx, item)
		if err != nil {
			return fmt.Errorf("unable to request video thumbnailer: %s", err)
		}

		file = resp.Body
	}

	a.increaseMetric("requested")

	if file == nil {
		file, err = a.storageApp.ReaderFrom(item.Pathname) // will be closed by `.Send`
		if err != nil {
			return err
		}
	}

	r, err := req.Post(a.imageURL).Build(ctx, file)
	if err != nil {
		return fmt.Errorf("unable to create request: %s", err)
	}

	if !item.IsVideo() {
		r.ContentLength = item.Size
	}

	resp, err = request.DoWithClient(thumbnailClient, r)
	if err != nil {
		return err
	}

	thumbnailPath := getThumbnailPath(item)
	if err := a.storageApp.CreateDir(filepath.Dir(thumbnailPath)); err != nil {
		return fmt.Errorf("unable to create directory: %s", err)
	}

	writer, err := a.storageApp.WriterTo(thumbnailPath)
	if err != nil {
		return fmt.Errorf("unable to get writer: %s", err)
	}

	if _, err := io.Copy(writer, resp.Body); err != nil {
		return err
	}

	a.increaseMetric("saved")

	return nil
}

func (a App) requestVith(ctx context.Context, item provider.StorageItem) (*http.Response, error) {
	if a.directAccess {
		return request.New().WithClient(thumbnailClient).Get(fmt.Sprintf("%s%s", a.videoURL, item.Pathname)).Send(ctx, nil)
	}

	return provider.SendLargeFile(ctx, a.storageApp, item, thumbnailClient, a.videoURL)
}
