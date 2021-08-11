package webhook

import (
	"context"

	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/httputils/v4/pkg/logger"
	"github.com/ViBiOh/httputils/v4/pkg/request"
)

// EventConsumer handle event pushed to the event bus
func (a *App) EventConsumer(e provider.Event) {
	if !a.Enabled() {
		return
	}

	a.mutex.RLock()
	defer a.mutex.RUnlock()

	for _, webhook := range a.webhooks {
		if !webhook.Match(e) {
			continue
		}

		req := request.New().Post(webhook.URL)
		for key, val := range webhook.Headers {
			req.Header(key, val)
		}

		resp, err := req.JSON(context.Background(), e)
		if err != nil {
			logger.Error("error while sending webhook: %s", err)
			continue
		}

		if err := resp.Body.Close(); err != nil {
			logger.Error("unable to close response body: %s", err)
		}
	}
}
