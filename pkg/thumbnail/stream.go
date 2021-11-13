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

	a.increaseMetric("video", "headers")

	resp, err := a.videoRequest.Method(http.MethodHead).Path(item.Pathname).Send(ctx, nil)
	if err != nil {
		return false, fmt.Errorf("unable to retrieve metadata: %s", err)
	}

	rawBitrate := resp.Header.Get("X-Vith-Bitrate")
	if len(rawBitrate) == 0 {
		return false, nil
	}

	bitrate, err := strconv.ParseUint(rawBitrate, 10, 64)
	if err != nil {
		return false, fmt.Errorf("unable to parse bitrate: %s", err)
	}

	if err := request.DiscardBody(resp.Body); err != nil {
		return false, fmt.Errorf("unable to discard body: %s", err)
	}

	logger.WithField("item", item.Pathname).Debug("Bitrate is %s", bitrate)

	return bitrate >= a.minBitrate, nil
}

func (a App) generateStream(ctx context.Context, item provider.StorageItem) error {
	a.increaseMetric("video", "stream")

	input := item.Pathname
	output := path.Dir(getStreamPath(item))

	payload, err := json.Marshal(map[string]string{
		"input":  item.Pathname,
		"output": path.Dir(getStreamPath(item)),
	})
	if err != nil {
		return fmt.Errorf("unable to marshal stream request: %s", err)
	}

	if a.amqpClient != nil {
		if err := a.amqpClient.Publish(amqp.Publishing{
			ContentType: "application/json",
			Body:        payload,
		}, a.amqpExchange, a.amqpStreamRoutingKey); err != nil {
			return fmt.Errorf("unable to publish amqp message: %s", err)
		}

		return nil
	}

	resp, err := a.videoRequest.Method(http.MethodPut).Path(fmt.Sprintf("%s?output=%s", input, url.QueryEscape(output))).Send(ctx, nil)
	if err != nil {
		return fmt.Errorf("unable to send request: %s", err)
	}

	if err := request.DiscardBody(resp.Body); err != nil {
		return fmt.Errorf("unable to discard body: %s", err)
	}

	return nil
}

func (a App) renameStream(ctx context.Context, old, new provider.StorageItem) error {
	a.increaseMetric("video", "rename")

	resp, err := a.videoRequest.Method(http.MethodPatch).Path(fmt.Sprintf("%s?to=%s", getStreamPath(old), url.QueryEscape(getStreamPath(new)))).Send(ctx, nil)
	if err != nil {
		return fmt.Errorf("unable to send request: %s", err)
	}

	if err := request.DiscardBody(resp.Body); err != nil {
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

	if err := request.DiscardBody(resp.Body); err != nil {
		return fmt.Errorf("unable to discard body: %s", err)
	}

	return nil
}
