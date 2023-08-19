package thumbnail

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	absto "github.com/ViBiOh/absto/pkg/model"
	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/httputils/v4/pkg/request"
	vith "github.com/ViBiOh/vith/pkg/model"
)

const (
	defaultTimeout = time.Minute * 2
)

func (a App) generate(ctx context.Context, item absto.Item, scale uint64) (err error) {
	ctx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()

	itemType := typeOfItem(item)

	var resp *http.Response
	resp, err = a.requestVith(ctx, item, scale)
	if err != nil {
		a.increaseMetric(ctx, itemType.String(), "error")
		return fmt.Errorf("request thumbnailer: %w", err)
	}

	if resp == nil {
		return nil
	}

	defer func() {
		if closeErr := request.DiscardBody(resp.Body); closeErr != nil {
			err = errors.Join(err, fmt.Errorf("close: %w", closeErr))
		}
	}()

	if resp.StatusCode == http.StatusNoContent {
		return nil
	}

	err = provider.WriteToStorage(ctx, a.storageApp, a.PathForScale(item, scale), resp.ContentLength, resp.Body)
	if err == nil {
		a.increaseMetric(ctx, itemType.String(), "save")
	}

	return err
}

func (a App) requestVith(ctx context.Context, item absto.Item, scale uint64) (*http.Response, error) {
	itemType := typeOfItem(item)
	outputName := a.PathForScale(item, scale)

	if a.amqpClient != nil {
		a.increaseMetric(ctx, itemType.String(), "publish")
		return nil, a.amqpClient.PublishJSON(ctx, vith.NewRequest(item.Pathname, outputName, itemType, scale), a.amqpExchange, a.amqpThumbnailRoutingKey)
	}

	a.increaseMetric(ctx, itemType.String(), "request")

	if a.directAccess {
		return a.vithRequest.Method(http.MethodGet).Path("%s?type=%s&scale=%d&output=%s", item.Pathname, itemType, scale, outputName).Send(ctx, nil)
	}

	return provider.SendLargeFile(ctx, a.storageApp, item, a.vithRequest.Method(http.MethodPost).Path("?type=%s&scale=%d", itemType, scale))
}
