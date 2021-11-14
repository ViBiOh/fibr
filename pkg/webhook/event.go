package webhook

import (
	"context"
	"strconv"

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

		resp, err := request.New().Post(webhook.URL).Header("User-Agent", "fibr-webhook").WithSignatureAuthorization("fibr", a.hmacSecret).JSON(context.Background(), e)
		if resp != nil {
			a.increaseMetric(strconv.Itoa(resp.StatusCode))
		}

		if err != nil {
			logger.Error("error while sending webhook: %s", err)
		} else if err := request.DiscardBody(resp.Body); err != nil {
			logger.Error("unable to discard body: %s", err)
		}
	}
}
