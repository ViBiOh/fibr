package thumbnail

import (
	"context"
	"fmt"
	"net/http"
	"time"

	absto "github.com/ViBiOh/absto/pkg/model"
	"github.com/ViBiOh/fibr/pkg/provider"
	httpModel "github.com/ViBiOh/httputils/v4/pkg/model"
	"github.com/ViBiOh/httputils/v4/pkg/request"
	vith "github.com/ViBiOh/vith/pkg/model"
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

	if resp.StatusCode == http.StatusNoContent {
		return nil
	}

	err = provider.WriteToStorage(a.storageApp, getThumbnailPath(item), resp.Body)
	if err == nil {
		a.increaseMetric(itemType.String(), "save")
	}

	return err
}

func (a App) requestVith(ctx context.Context, item absto.Item) (*http.Response, error) {
	itemType := typeOfItem(item)
	outputName := getThumbnailPath(item)

	if a.amqpClient != nil {
		a.increaseMetric(itemType.String(), "publish")
		return nil, a.amqpClient.PublishJSON(vith.NewRequest(item.Pathname, outputName, itemType), a.amqpExchange, a.amqpThumbnailRoutingKey)
	}

	a.increaseMetric(itemType.String(), "request")

	if a.directAccess {
		return a.vithRequest.Method(http.MethodGet).Path(fmt.Sprintf("%s?type=%s&to=%s", item.Pathname, itemType, outputName)).Send(ctx, nil)
	}

	return provider.SendLargeFile(ctx, a.storageApp, item, a.vithRequest.Method(http.MethodPost).Path(fmt.Sprintf("?type=%s", itemType)))
}
