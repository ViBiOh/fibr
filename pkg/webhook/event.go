package webhook

import (
	"context"
	"fmt"
	"net/url"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/httputils/v4/pkg/logger"
	"github.com/ViBiOh/httputils/v4/pkg/request"
)

// EventConsumer handle event pushed to the event bus
func (a *App) EventConsumer(event provider.Event) {
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
		case provider.Telegram:
			statusCode, err = a.telegramHandle(context.Background(), webhook, event)
		default:
			logger.Warn("unknown kind `%d` for webhook", webhook.Kind)
		}

		a.increaseMetric(strconv.Itoa(statusCode))

		if err != nil {
			logger.Error("error while sending webhook: %s", err)
		}
	}

	go func() {
		if event.Type == provider.DeleteEvent {
			if err := a.deleteItem(event.Item); err != nil {
				logger.Error("unable to delete webhooks for item: %s", err)
			}
		}
	}()
}

func send(ctx context.Context, id string, req request.Request, payload interface{}) (int, error) {
	resp, err := req.JSON(ctx, payload)
	if err != nil {
		return 0, fmt.Errorf("unable to send webhook with id `%s`: %s", id, err)
	}

	if err = request.DiscardBody(resp.Body); err != nil {
		return resp.StatusCode, fmt.Errorf("unable to discard body for webhook with id `%s`: %s", id, err)
	}

	return resp.StatusCode, nil
}

func (a *App) rawHandle(ctx context.Context, webhook provider.Webhook, event provider.Event) (int, error) {
	return send(ctx, webhook.ID, request.Post(webhook.URL).Header("User-Agent", "fibr-webhook").WithSignatureAuthorization("fibr", a.hmacSecret), event)
}

func (a *App) discordHandle(ctx context.Context, webhook provider.Webhook, event provider.Event) (int, error) {
	if event.Type != provider.UploadEvent {
		return send(ctx, webhook.ID, request.Post(webhook.URL), discordPayload{
			Content: a.eventText(event),
		})
	}

	url := event.GetURL()
	title := path.Base(path.Dir(event.Item.Pathname))
	if title == "/" {
		title = "fibr"
	}

	embed := discordEmbed{
		Title:       title,
		Description: "💾 A file has been uploaded",
		URL:         url + "?browser",
		Fields: []discordField{{
			Name:   "item",
			Value:  event.Item.Name,
			Inline: true,
		}},
	}

	if a.thumbnailApp.CanHaveThumbnail(event.Item) {
		embed.Thumbnail = &discordContent{
			URL: url + "?thumbnail",
		}
		embed.Image = &discordContent{
			URL: url + "?thumbnail",
		}
	}

	return send(ctx, webhook.ID, request.Post(webhook.URL), discordPayload{
		Embeds: []discordEmbed{embed},
	})
}

func (a *App) slackHandle(ctx context.Context, webhook provider.Webhook, event provider.Event) (int, error) {
	if event.Type != provider.UploadEvent {
		return send(ctx, webhook.ID, request.Post(webhook.URL), slackPayload{
			Text: a.eventText(event),
		})
	}

	url := event.GetURL()
	title := path.Base(path.Dir(event.Item.Pathname))
	if title == "/" {
		title = "fibr"
	}

	section := slackSection{
		Type:   "section",
		Text:   newText("💾 A file has been uploaded"),
		Fields: []slackText{newText(fmt.Sprintf("*item*\n%s", event.Item.Name))},
	}

	if a.thumbnailApp.CanHaveThumbnail(event.Item) {
		section.Accessory = &slackImage{
			Type: "image",
			URL:  url + "?thumbnail",
			Alt:  fmt.Sprintf("Thumbnail of %s", event.Item.Name),
		}
	}

	return send(ctx, webhook.ID, request.Post(webhook.URL), slackPayload{
		Text: a.eventText(event),
		Blocks: []slackSection{
			{
				Type: "section",
				Text: newText(fmt.Sprintf("*<%s?browser|%s>*", url, title)),
			},
			section,
		},
	})
}

func (a *App) telegramHandle(ctx context.Context, webhook provider.Webhook, event provider.Event) (int, error) {
	return send(ctx, webhook.ID, request.Post(fmt.Sprintf("%s&message=%s", webhook.URL, url.QueryEscape(a.eventText(event)))), nil)
}

func (a *App) eventText(event provider.Event) string {
	switch event.Type {
	case provider.AccessEvent:
		return a.accessEvent(event)
	case provider.CreateDir:
		return fmt.Sprintf("🗂 A directory `%s` has been created: %s", event.Item.Name, event.GetURL())
	case provider.UploadEvent:
		return fmt.Sprintf("💾 A file has been uploaded: %s?browser", event.GetURL())
	case provider.RenameEvent:
		return fmt.Sprintf("➡️ `%s` has been renamed to `%s`", event.Item.Pathname, event.New.Pathname)
	case provider.DeleteEvent:
		return fmt.Sprintf("❌ `%s` has been deleted : %s", event.Item.Name, event.GetURL())
	case provider.StartEvent:
		return fmt.Sprintf("🚀 Fibr starts routine for path `%s`", event.Item.Pathname)
	default:
		return fmt.Sprintf("🙄 Event `%s` occurred on `%s`", event.Type, event.Item.Name)
	}
}

func (a *App) accessEvent(event provider.Event) string {
	content := strings.Builder{}
	content.WriteString(fmt.Sprintf("💻 Someone connected to Fibr from %s at %s", a.rendererApp.PublicURL(event.URL), event.Time.Format(time.RFC3339)))

	if len(event.Metadata) > 0 {
		content.WriteString("\n```\n")

		for key, value := range event.Metadata {
			content.WriteString(fmt.Sprintf("%s: %s\n", key, value))
		}

		content.WriteString("```")
	}

	return content.String()
}

func (a *App) deleteItem(item provider.StorageItem) error {
	return a.Exclusive(context.Background(), a.amqpExclusiveRoutingKey, semaphoreDuration, func(_ context.Context) error {
		for id, webhook := range a.webhooks {
			if webhook.Pathname == item.Pathname {
				if err := a.delete(id); err != nil {
					return fmt.Errorf("unable to delete webhook `%s`: %s", id, err)
				}
			}
		}

		return nil
	})
}
