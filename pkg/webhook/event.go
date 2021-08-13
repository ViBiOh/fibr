package webhook

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
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

		req := request.New().Post(webhook.URL)

		var resp *http.Response
		var err error

		if len(a.hmacSecret) == 0 {
			resp, err = req.JSON(context.Background(), e)
		} else {
			resp, err = a.sendWithHmac(context.Background(), req, e)
		}

		if resp != nil {
			a.increaseMetric(strconv.Itoa(resp.StatusCode))
		}
		if err != nil {
			logger.Error("error while sending webhook: %s", err)

			continue
		}

		if err := resp.Body.Close(); err != nil {
			logger.Error("unable to close response body: %s", err)
		}
	}
}

func (a *App) sendWithHmac(ctx context.Context, req *request.Request, event provider.Event) (*http.Response, error) {
	hasher := hmac.New(sha256.New, []byte(a.hmacSecret))

	payload, err := json.Marshal(event)
	if err != nil {
		return nil, fmt.Errorf("unable to marshal: %s", err)
	}

	hasher.Write(payload)
	req.Header("X-Fibr-Signature", hex.EncodeToString(hasher.Sum(nil)))
	req.ContentJSON()
	return req.Send(ctx, bytes.NewReader(payload))
}
