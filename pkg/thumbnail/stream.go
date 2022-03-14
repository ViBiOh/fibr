package thumbnail

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"

	absto "github.com/ViBiOh/absto/pkg/model"
	"github.com/ViBiOh/httputils/v4/pkg/logger"
	"github.com/ViBiOh/httputils/v4/pkg/request"
	"github.com/ViBiOh/vith/pkg/model"
)

// HasStream checks if given item has a streamable version
func (a App) HasStream(ctx context.Context, item absto.Item) bool {
	_, err := a.storageApp.Info(ctx, getStreamPath(item))
	return err == nil
}

func (a App) handleVithResponse(err error, body io.ReadCloser) error {
	if err != nil {
		a.increaseMetric("stream", "error")
		return fmt.Errorf("unable to send request: %s", err)
	}

	if err := request.DiscardBody(body); err != nil {
		return fmt.Errorf("unable to discard body: %s", err)
	}

	return nil
}

func (a App) shouldGenerateStream(ctx context.Context, item absto.Item) (bool, error) {
	if !a.directAccess {
		return false, nil
	}

	a.increaseMetric("stream", "bitrate")

	resp, err := a.vithRequest.Method(http.MethodHead).Path(fmt.Sprintf("%s?type=%s", item.Pathname, typeOfItem(item))).Send(ctx, nil)
	if err != nil {
		a.increaseMetric("stream", "error")
		return false, fmt.Errorf("unable to retrieve metadata: %s", err)
	}

	rawBitrate := resp.Header.Get("X-Vith-Bitrate")
	if len(rawBitrate) == 0 {
		return false, nil
	}

	bitrate, err := strconv.ParseUint(rawBitrate, 10, 64)
	if err != nil {
		a.increaseMetric("stream", "error")
		return false, fmt.Errorf("unable to parse bitrate: %s", err)
	}

	if err := request.DiscardBody(resp.Body); err != nil {
		return false, fmt.Errorf("unable to discard body: %s", err)
	}

	logger.WithField("item", item.Pathname).Debug("Bitrate is %s", bitrate)

	return bitrate >= a.minBitrate, nil
}

func (a App) generateStream(ctx context.Context, item absto.Item) error {
	input := item.Pathname
	output := getStreamPath(item)

	req := model.NewRequest(input, getStreamPath(item), typeOfItem(item), SmallSize)

	if a.amqpClient != nil {
		a.increaseMetric("stream", "publish")

		err := a.amqpClient.PublishJSON(req, a.amqpExchange, a.amqpStreamRoutingKey)
		if err != nil {
			a.increaseMetric("stream", "error")
		}

		return err
	}

	a.increaseMetric("stream", "request")

	resp, err := a.vithRequest.Method(http.MethodPut).Path(fmt.Sprintf("%s?output=%s", input, url.QueryEscape(output))).Send(ctx, nil)
	return a.handleVithResponse(err, resp.Body)
}

func (a App) renameStream(ctx context.Context, old, new absto.Item) error {
	a.increaseMetric("stream", "rename")

	resp, err := a.vithRequest.Method(http.MethodPatch).Path(fmt.Sprintf("%s?to=%s&type=%s", getStreamPath(old), url.QueryEscape(getStreamPath(new)), typeOfItem(old))).Send(ctx, nil)
	return a.handleVithResponse(err, resp.Body)
}

func (a App) deleteStream(ctx context.Context, item absto.Item) error {
	a.increaseMetric("stream", "delete")

	resp, err := a.vithRequest.Method(http.MethodDelete).Path(fmt.Sprintf("%s?type=%s", getStreamPath(item), typeOfItem(item))).Send(ctx, nil)
	return a.handleVithResponse(err, resp.Body)
}
