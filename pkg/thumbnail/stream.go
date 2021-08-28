package thumbnail

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strconv"

	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/httputils/v4/pkg/logger"
)

// HasStream checks if given item has a streamable version
func (a App) HasStream(item provider.StorageItem) bool {
	_, err := a.storageApp.Info(getStreamPath(item))
	return err == nil
}

func (a App) shouldGenerateStream(ctx context.Context, item provider.StorageItem) (bool, error) {
	if !a.directAccess {
		return false, nil
	}

	a.increaseMetric("video", "headers")

	resp, err := a.videoRequest.Method(http.MethodHead).Path(item.Pathname).Send(ctx, nil)
	if err != nil {
		return false, fmt.Errorf("unable to retrieve metadata: %s", err)
	}

	defer func() {
		if err := resp.Body.Close(); err != nil {
			logger.Error("unable to close body: %s", err)
		}
	}()
	if _, err := io.Copy(io.Discard, resp.Body); err != nil {
		return false, fmt.Errorf("unable to discard body: %s", err)
	}

	rawBitrate := resp.Header.Get("X-Vith-Bitrate")
	if len(rawBitrate) == 0 {
		return false, nil
	}

	bitrate, err := strconv.ParseUint(rawBitrate, 10, 64)
	if err != nil {
		return false, fmt.Errorf("unable to parse bitrate: %s", err)
	}

	return bitrate >= a.minBitrate, nil
}

func (a App) generateStream(ctx context.Context, item provider.StorageItem) error {
	a.increaseMetric("video", "stream")

	resp, err := a.videoRequest.Method(http.MethodPut).Path(fmt.Sprintf("%s?output=%s", item.Pathname, url.QueryEscape(path.Dir(getStreamPath(item))))).Send(ctx, nil)
	if err != nil {
		return fmt.Errorf("unable to send request: %s", err)
	}

	defer func() {
		if err := resp.Body.Close(); err != nil {
			logger.Error("unable to close generate stream body: %s", err)
		}
	}()
	if _, err := io.Copy(io.Discard, resp.Body); err != nil {
		return fmt.Errorf("unable to discard body: %s", err)
	}

	return nil
}

func (a App) deleteStream(ctx context.Context, item provider.StorageItem) error {
	a.increaseMetric("video", "delete")

	resp, err := a.videoRequest.Method(http.MethodDelete).Path(getStreamPath(item)).Send(ctx, nil)
	if err != nil {
		return fmt.Errorf("unable to send request: %s", err)
	}

	defer func() {
		if err := resp.Body.Close(); err != nil {
			logger.Error("unable to close delete stream body: %s", err)
		}
	}()
	if _, err := io.Copy(io.Discard, resp.Body); err != nil {
		return fmt.Errorf("unable to discard body: %s", err)
	}

	return nil
}
