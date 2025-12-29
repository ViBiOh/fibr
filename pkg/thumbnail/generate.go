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
	vignet "github.com/ViBiOh/vignet/pkg/model"
)

const (
	defaultTimeout = time.Minute * 2
	quickTimeout   = time.Second * 30
)

func (s Service) generate(ctx context.Context, item absto.Item, scale uint64) (err error) {
	ctx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()

	itemType := typeOfItem(item)

	var resp *http.Response
	resp, err = s.requestVignet(ctx, item, itemType, scale)
	if err != nil {
		s.increaseMetric(ctx, itemType.String(), "error")
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

	err = provider.WriteToStorage(ctx, s.storage, s.PathForScale(item, scale), resp.ContentLength, resp.Body)
	if err == nil {
		s.increaseMetric(ctx, itemType.String(), "save")
	}

	return err
}

func (s Service) requestVignet(ctx context.Context, item absto.Item, itemType vignet.ItemType, scale uint64) (*http.Response, error) {
	outputName := s.PathForScale(item, scale)

	if s.amqpClient != nil {
		s.increaseMetric(ctx, itemType.String(), "publish")
		return nil, s.amqpClient.PublishJSON(ctx, vignet.NewRequest(item.Pathname, outputName, itemType, scale), s.amqpExchange, s.amqpThumbnailRoutingKey)
	}

	s.increaseMetric(ctx, itemType.String(), "request")

	if s.directAccess {
		return s.vignetRequest.Method(http.MethodGet).Path("%s?type=%s&scale=%d&output=%s", item.Pathname, itemType, scale, outputName).Send(ctx, nil)
	}

	return provider.SendLargeFile(ctx, s.storage, item, s.vignetRequest.Method(http.MethodPost).Path("?type=%s&scale=%d", itemType, scale))
}
