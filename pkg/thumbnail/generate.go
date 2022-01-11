package thumbnail

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"time"

	absto "github.com/ViBiOh/absto/pkg/model"
	"github.com/ViBiOh/fibr/pkg/provider"
	httpModel "github.com/ViBiOh/httputils/v4/pkg/model"
	"github.com/ViBiOh/httputils/v4/pkg/request"
	"github.com/ViBiOh/vith/pkg/model"
)

const (
	defaultTimeout = time.Minute * 2
)

func (a App) generate(item absto.Item) (err error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	itemType := typeOfItem(item)

	var resp *http.Response
	resp, err = a.requestVith(ctx, item)
	if err != nil {
		a.increaseMetric(itemType.String(), "error")
		return fmt.Errorf("unable to request thumbnailer: %s", err)
	}

	if resp == nil {
		return nil
	}

	defer func() {
		if closeErr := request.DiscardBody(resp.Body); closeErr != nil {
			err = httpModel.WrapError(err, fmt.Errorf("unable to close: %s", closeErr))
		}
	}()

	thumbnailPath := getThumbnailPath(item)
	if err = a.storageApp.CreateDir(filepath.Dir(thumbnailPath)); err != nil {
		return fmt.Errorf("unable to create directory: %s", err)
	}

	var writer io.WriteCloser
	writer, err = a.storageApp.WriterTo(thumbnailPath)
	if err != nil {
		return fmt.Errorf("unable to get writer: %w", err)
	}

	defer func() {
		if closeErr := writer.Close(); closeErr != nil {
			err = httpModel.WrapError(err, fmt.Errorf("unable to close: %s", closeErr))
		}
	}()

	buffer := provider.BufferPool.Get().(*bytes.Buffer)
	defer provider.BufferPool.Put(buffer)

	if _, err = io.CopyBuffer(writer, resp.Body, buffer.Bytes()); err != nil {
		err = fmt.Errorf("unable to copy response: %s", err)
	}

	a.increaseMetric(itemType.String(), "save")

	return err
}

func (a App) requestVith(ctx context.Context, item absto.Item) (*http.Response, error) {
	itemType := typeOfItem(item)

	if a.amqpClient != nil {
		a.increaseMetric(itemType.String(), "publish")

		err := a.amqpClient.PublishJSON(model.NewRequest(item.Pathname, getThumbnailPath(item), itemType), a.amqpExchange, a.amqpThumbnailRoutingKey)
		if err != nil {
			a.increaseMetric(itemType.String(), "error")
		}

		return nil, err
	}

	a.increaseMetric(itemType.String(), "request")

	if a.directAccess {
		return a.vithRequest.Method(http.MethodGet).Path(fmt.Sprintf("%s?type=%s", item.Pathname, itemType)).Send(ctx, nil)
	}

	return provider.SendLargeFile(ctx, a.storageApp, item, a.vithRequest.Method(http.MethodPost).Path(fmt.Sprintf("?type=%s", itemType)))
}
