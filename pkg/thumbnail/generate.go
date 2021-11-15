package thumbnail

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"time"

	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/httputils/v4/pkg/request"
	"github.com/streadway/amqp"
)

const (
	defaultTimeout = time.Minute * 2
)

func (a App) generate(item provider.StorageItem) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	var (
		file io.ReadCloser
		resp *http.Response
		err  error
	)

	if item.IsVideo() {
		resp, err = a.requestVith(ctx, item)
		if err != nil {
			return fmt.Errorf("unable to request video thumbnailer: %s", err)
		}

		if resp == nil {
			return nil
		}
	} else {
		file, err = a.storageApp.ReaderFrom(item.Pathname) // will be closed by `.Send`
		if err != nil {
			return err
		}

		r, err := a.imageRequest.Build(ctx, file)
		if err != nil {
			return fmt.Errorf("unable to create request: %s", err)
		}

		a.increaseMetric("image", "requested")

		r.ContentLength = item.Size
		resp, err = request.DoWithClient(provider.SlowClient, r)
		if err != nil {
			return fmt.Errorf("unable to request image thumbnailer: %s", err)
		}
	}

	thumbnailPath := getThumbnailPath(item)
	if err := a.storageApp.CreateDir(filepath.Dir(thumbnailPath)); err != nil {
		return fmt.Errorf("unable to create directory: %s", err)
	}

	writer, err := a.storageApp.WriterTo(thumbnailPath)
	if err != nil {
		return fmt.Errorf("unable to get writer: %s", err)
	}

	buffer := provider.BufferPool.Get().(*bytes.Buffer)
	defer provider.BufferPool.Put(buffer)

	if _, err = io.CopyBuffer(writer, resp.Body, buffer.Bytes()); err != nil {
		err = fmt.Errorf("unable to copy response: %s", err)
	}

	if closeErr := resp.Body.Close(); closeErr != nil {
		if err != nil {
			return fmt.Errorf("%s: %w", err, closeErr)
		}
		err = fmt.Errorf("unable to close body response: %s", err)
	}

	if closeErr := writer.Close(); closeErr != nil {
		if err != nil {
			return fmt.Errorf("%s: %w", err, closeErr)
		}
		return fmt.Errorf("unable to close writer: %s", err)
	}

	a.increaseMetric("thumbnail", "saved")

	return err
}

func (a App) requestVith(ctx context.Context, item provider.StorageItem) (*http.Response, error) {
	if a.amqpClient != nil {
		payload, err := json.Marshal(map[string]string{
			"input":  item.Pathname,
			"output": getThumbnailPath(item),
		})
		if err != nil {
			return nil, fmt.Errorf("unable to marshal video thumbnail amqp message: %s", err)
		}

		a.increaseMetric("video", "published")

		if err := a.amqpClient.Publish(amqp.Publishing{
			ContentType: "application/json",
			Body:        payload,
		}, a.amqpExchange, a.amqpVideoThumbnailRoutingKey); err != nil {
			return nil, fmt.Errorf("unable to publish video thumbnail amqp message: %s", err)
		}

		return nil, nil
	}

	a.increaseMetric("video", "requested")

	if a.directAccess {
		return a.videoRequest.Method(http.MethodGet).Path(item.Pathname).Send(ctx, nil)
	}

	return provider.SendLargeFile(ctx, a.storageApp, item, a.videoRequest.Method(http.MethodPost))
}
