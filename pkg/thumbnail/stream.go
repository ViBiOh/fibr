package thumbnail

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"

	absto "github.com/ViBiOh/absto/pkg/model"
	"github.com/ViBiOh/httputils/v4/pkg/request"
	"github.com/ViBiOh/vith/pkg/model"
)

func (s Service) HasStream(ctx context.Context, item absto.Item) bool {
	_, err := s.Info(ctx, getStreamPath(item))
	return err == nil
}

func (s Service) handleVithResponse(ctx context.Context, err error, body io.ReadCloser) error {
	if err != nil {
		s.increaseMetric(ctx, "stream", "error")
		return fmt.Errorf("send request: %w", err)
	}

	if err := request.DiscardBody(body); err != nil {
		return fmt.Errorf("discard body: %w", err)
	}

	return nil
}

func (s Service) shouldGenerateStream(ctx context.Context, item absto.Item) (bool, error) {
	if !s.directAccess {
		return false, nil
	}

	s.increaseMetric(ctx, "stream", "bitrate")

	resp, err := s.vithRequest.Method(http.MethodHead).Path("%s?type=%s", item.Pathname, typeOfItem(item)).Send(ctx, nil)
	if err != nil {
		s.increaseMetric(ctx, "stream", "error")
		return false, fmt.Errorf("retrieve metadata: %w", err)
	}

	rawBitrate := resp.Header.Get("X-Vith-Bitrate")
	if len(rawBitrate) == 0 {
		return false, nil
	}

	bitrate, err := strconv.ParseUint(rawBitrate, 10, 64)
	if err != nil {
		s.increaseMetric(ctx, "stream", "error")
		return false, fmt.Errorf("parse bitrate: %w", err)
	}

	if err := request.DiscardBody(resp.Body); err != nil {
		return false, fmt.Errorf("discard body: %w", err)
	}

	slog.Debug("Bitrate", "bitrate", bitrate, "item", item.Pathname)

	return bitrate >= s.minBitrate, nil
}

func (s Service) generateStream(ctx context.Context, item absto.Item) error {
	input := item.Pathname
	output := getStreamPath(item)

	req := model.NewRequest(input, getStreamPath(item), typeOfItem(item), SmallSize)

	if s.amqpClient != nil {
		s.increaseMetric(ctx, "stream", "publish")

		err := s.amqpClient.PublishJSON(ctx, req, s.amqpExchange, s.amqpStreamRoutingKey)
		if err != nil {
			s.increaseMetric(ctx, "stream", "error")
		}

		return err
	}

	s.increaseMetric(ctx, "stream", "request")

	resp, err := s.vithRequest.Method(http.MethodPut).Path("%s?output=%s", input, url.QueryEscape(output)).Send(ctx, nil)
	return s.handleVithResponse(ctx, err, resp.Body)
}

func (s Service) renameStream(ctx context.Context, old, new absto.Item) error {
	s.increaseMetric(ctx, "stream", "rename")

	resp, err := s.vithRequest.Method(http.MethodPatch).Path("%s?to=%s&type=%s", getStreamPath(old), url.QueryEscape(getStreamPath(new)), typeOfItem(old)).Send(ctx, nil)
	return s.handleVithResponse(ctx, err, resp.Body)
}

func (s Service) deleteStream(ctx context.Context, item absto.Item) error {
	s.increaseMetric(ctx, "stream", "delete")

	resp, err := s.vithRequest.Method(http.MethodDelete).Path("%s?type=%s", getStreamPath(item), typeOfItem(item)).Send(ctx, nil)
	return s.handleVithResponse(ctx, err, resp.Body)
}
