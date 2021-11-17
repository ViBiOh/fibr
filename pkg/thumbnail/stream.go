package thumbnail

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"strconv"

	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/httputils/v4/pkg/logger"
	"github.com/ViBiOh/httputils/v4/pkg/request"
	"github.com/ViBiOh/vith/pkg/model"
	"github.com/streadway/amqp"
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

	a.increaseMetric("stream", "bitrate")

	resp, err := a.vithRequest.Method(http.MethodHead).Path(fmt.Sprintf("%s?type=%s", item.Pathname, typeOfItem(item))).Send(ctx, nil)
	if err != nil {
		a.increaseMetric("stream", "errror")
		return false, fmt.Errorf("unable to retrieve metadata: %s", err)
	}

	rawBitrate := resp.Header.Get("X-Vith-Bitrate")
	if len(rawBitrate) == 0 {
		return false, nil
	}

	bitrate, err := strconv.ParseUint(rawBitrate, 10, 64)
	if err != nil {
		a.increaseMetric("stream", "errror")
		return false, fmt.Errorf("unable to parse bitrate: %s", err)
	}

	if err := request.DiscardBody(resp.Body); err != nil {
		return false, fmt.Errorf("unable to discard body: %s", err)
	}

	logger.WithField("item", item.Pathname).Debug("Bitrate is %s", bitrate)

	return bitrate >= a.minBitrate, nil
}

func (a App) generateStream(ctx context.Context, item provider.StorageItem) error {
	input := item.Pathname
	output := path.Dir(getStreamPath(item))

	req := model.NewRequest(input, output, typeOfItem(item))

	if a.amqpClient != nil {
		a.increaseMetric("stream", "publish")

		payload, err := json.Marshal(req)
		if err != nil {
			a.increaseMetric("stream", "errror")
			return fmt.Errorf("unable to marshal stream amqp message: %s", err)
		}

		if err := a.amqpClient.Publish(amqp.Publishing{
			ContentType: "application/json",
			Body:        payload,
		}, a.amqpExchange, a.amqpStreamRoutingKey); err != nil {
			a.increaseMetric("stream", "error")
			return fmt.Errorf("unable to publish stream amqp message: %s", err)
		}

		return nil
	}

	a.increaseMetric("stream", "request")

	resp, err := a.vithRequest.Method(http.MethodPut).Path(fmt.Sprintf("%s?output=%s", input, url.QueryEscape(output))).Send(ctx, nil)
	if err != nil {
		a.increaseMetric("stream", "error")
		return fmt.Errorf("unable to send request: %s", err)
	}

	if err := request.DiscardBody(resp.Body); err != nil {
		return fmt.Errorf("unable to discard body: %s", err)
	}

	return nil
}

func (a App) renameStream(ctx context.Context, old, new provider.StorageItem) error {
	a.increaseMetric("stream", "rename")

	resp, err := a.vithRequest.Method(http.MethodPatch).Path(fmt.Sprintf("%s?to=%s&type=%s", getStreamPath(old), url.QueryEscape(getStreamPath(new)), typeOfItem(old))).Send(ctx, nil)
	if err != nil {
		a.increaseMetric("stream", "error")
		return fmt.Errorf("unable to send request: %s", err)
	}

	if err := request.DiscardBody(resp.Body); err != nil {
		return fmt.Errorf("unable to discard body: %s", err)
	}

	return nil
}

func (a App) deleteStream(ctx context.Context, item provider.StorageItem) error {
	a.increaseMetric("stream", "delete")

	resp, err := a.vithRequest.Method(http.MethodDelete).Path(fmt.Sprintf("%s?type=%s", getStreamPath(item), typeOfItem(item))).Send(ctx, nil)
	if err != nil {
		a.increaseMetric("stream", "error")
		return fmt.Errorf("unable to send request: %s", err)
	}

	if err := request.DiscardBody(resp.Body); err != nil {
		return fmt.Errorf("unable to discard body: %s", err)
	}

	return nil
}
