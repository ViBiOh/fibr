package webhook

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/httputils/v4/pkg/logger"
	"github.com/ViBiOh/httputils/v4/pkg/request"
)

type discordPayload struct {
	Content string `json:"content"`
}

type slackPayload struct {
	Text string `json:"text"`
}

// EventConsumer handle event pushed to the event bus
func (a *App) EventConsumer(event provider.Event) {
	if !a.Enabled() {
		return
	}

	a.RLock()
	defer a.RUnlock()

	for _, webhook := range a.webhooks {
		if !webhook.Match(event) {
			continue
		}

		var statusCode int
		var err error

		switch webhook.Kind {
		case provider.Raw:
			statusCode, err = a.rawHandle(context.Background(), webhook, event)
		case provider.Discord:
			statusCode, err = a.discordHandle(context.Background(), webhook, event)
		case provider.Slack:
			statusCode, err = a.slackHandle(context.Background(), webhook, event)
		default:
			logger.Warn("unknown kind `%d` for webhook", webhook.Kind)
		}

		a.increaseMetric(strconv.Itoa(statusCode))

		if err != nil {
			logger.Error("error while sending webhook: %s", err)
		}
	}
}

func send(ctx context.Context, req request.Request, payload interface{}) (int, error) {
	resp, err := req.JSON(ctx, payload)
	if err != nil {
		return 0, fmt.Errorf("unable to send discord webhook: %s", err)
	}

	if err = request.DiscardBody(resp.Body); err != nil {
		return resp.StatusCode, fmt.Errorf("unable to discard discord body: %s", err)
	}

	return resp.StatusCode, nil
}

func (a *App) rawHandle(ctx context.Context, webhook provider.Webhook, event provider.Event) (int, error) {
	return send(ctx, request.New().Post(webhook.URL).Header("User-Agent", "fibr-webhook").WithSignatureAuthorization("fibr", a.hmacSecret), event)
}

func (a *App) discordHandle(ctx context.Context, webhook provider.Webhook, event provider.Event) (int, error) {
	return send(ctx, request.New().Post(webhook.URL), discordPayload{
		Content: a.eventText(event),
	})
}

func (a *App) slackHandle(ctx context.Context, webhook provider.Webhook, event provider.Event) (int, error) {
	return send(ctx, request.New().Post(webhook.URL), slackPayload{
		Text: a.eventText(event),
	})
}

func (a *App) eventText(event provider.Event) string {
	switch event.Type {
	case provider.AccessEvent:
		return a.discordAccess(event)
	case provider.CreateDir:
		return fmt.Sprintf("🗂 A directory `%s` has been created: %s", event.Item.Name, a.rendererApp.PublicURL(event.URL))
	case provider.UploadEvent:
		return fmt.Sprintf("💾 A file has been uploaded: %s", a.rendererApp.PublicURL(event.URL))
	case provider.RenameEvent:
		return fmt.Sprintf("➡️ `%s` has been renamed to `%s`: %s", event.Item.Pathname, event.New.Pathname, a.rendererApp.PublicURL(event.URL))
	case provider.DeleteEvent:
		return fmt.Sprintf("❌ `%s` has been deleted : %s", event.Item.Name, a.rendererApp.PublicURL(event.URL))
	case provider.StartEvent:
		return fmt.Sprintf("🚀 Fibr starts routine for path `%s`", event.Item.Pathname)
	default:
		return fmt.Sprintf("🙄 Event `%s` occurred on `%s`", event.Type, event.Item.Name)
	}
}

func (a *App) discordAccess(event provider.Event) string {
	content := strings.Builder{}
	content.WriteString(fmt.Sprintf("💻 Someone connected to Fibr from %s at %s", a.rendererApp.PublicURL(event.URL), event.Time.Format(time.RFC3339)))

	if len(event.Metadata) > 0 {
		content.WriteString("```\n")

		for key, value := range event.Metadata {
			content.WriteString(fmt.Sprintf("%s: %s\n", key, value))
		}

		content.WriteString("```")
	}

	return content.String()
}
