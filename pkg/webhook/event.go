package webhook

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/ViBiOh/ChatPotte/discord"
	"github.com/ViBiOh/ChatPotte/slack"
	absto "github.com/ViBiOh/absto/pkg/model"
	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/httputils/v4/pkg/request"
)

func (s *Service) EventConsumer(ctx context.Context, event provider.Event) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	for _, webhook := range s.webhooks {
		if !webhook.Match(event) {
			continue
		}
		var statusCode int
		var err error

		switch webhook.Kind {
		case provider.Raw:
			statusCode, err = s.rawHandle(ctx, webhook, event)

		case provider.Discord:
			statusCode, err = s.discordHandle(ctx, webhook, event)

		case provider.Slack:
			statusCode, err = s.slackHandle(ctx, webhook, event)

		case provider.Telegram:
			statusCode, err = s.telegramHandle(ctx, webhook, event)

		default:
			slog.WarnContext(ctx, "unknown kind for webhook", "kind", webhook.Kind)
		}

		s.increaseMetric(ctx, strconv.Itoa(statusCode))

		if err != nil {
			slog.ErrorContext(ctx, "error while sending webhook", "error", err)
		}
	}

	if event.Type == provider.DeleteEvent {
		// Fire a goroutine to release the mutex lock
		go func() {
			if err := s.deleteItem(ctx, event.Item); err != nil {
				slog.ErrorContext(ctx, "delete webhooks for item", "error", err)
			}
		}()
	}
}

func send(ctx context.Context, id string, req request.Request, payload any) (int, error) {
	resp, err := req.JSON(ctx, payload)
	if err != nil {
		return 0, fmt.Errorf("send webhook with id `%s`: %w", id, err)
	}

	if err = request.DiscardBody(resp.Body); err != nil {
		return resp.StatusCode, fmt.Errorf("discard body for webhook with id `%s`: %w", id, err)
	}

	return resp.StatusCode, nil
}

func (s *Service) rawHandle(ctx context.Context, webhook provider.Webhook, event provider.Event) (int, error) {
	return send(ctx, webhook.ID, request.Post(webhook.URL).Header("User-Agent", "fibr-webhook").WithSignatureAuthorization("fibr", s.hmacSecret), event)
}

func (s *Service) discordHandle(ctx context.Context, webhook provider.Webhook, event provider.Event) (int, error) {
	if event.Type != provider.UploadEvent && event.Type != provider.RenameEvent && event.Type != provider.DescriptionEvent {
		return send(ctx, webhook.ID, request.Post(webhook.URL), discord.NewDataResponse(s.eventText(event)))
	}

	title := path.Base(path.Dir(event.Item.Pathname))
	if title == "/" {
		title = "fibr"
	}

	var contentURL, description string
	fields := []discord.Field{discord.NewField("item", event.GetName())}

	switch event.Type {
	case provider.UploadEvent:
		description = "💾 A file has been uploaded"
		contentURL = event.BrowserURL()
	case provider.RenameEvent:
		description = "✏️ An item has been renamed"
		fields = append(fields, discord.NewField("to", event.GetTo()))
		contentURL = event.BrowserURL()
	case provider.DescriptionEvent:
		description = "💬 " + event.Metadata["description"]
		contentURL = event.StoryURL(event.Item.ID)
	}

	embed := discord.Embed{
		Title:       title,
		Description: description,
		URL:         contentURL,
		Fields:      fields,
	}

	if s.thumbnail.CanHaveThumbnail(event.Item) {
		thumbnailURL := event.GetURL() + "?thumbnail"

		if _, ok := provider.VideoExtensions[event.Item.Extension]; ok {
			thumbnailURL += "&scale=large"
		}

		embed.Thumbnail = discord.NewImage(thumbnailURL)
	}

	return send(ctx, webhook.ID, request.Post(webhook.URL), discord.NewDataResponse("").AddEmbed(embed))
}

func (s *Service) slackHandle(ctx context.Context, webhook provider.Webhook, event provider.Event) (int, error) {
	if event.Type != provider.UploadEvent && event.Type != provider.RenameEvent && event.Type != provider.DescriptionEvent {
		return send(ctx, webhook.ID, request.Post(webhook.URL), slack.NewResponse(s.eventText(event)))
	}

	title := path.Base(path.Dir(event.Item.Pathname))
	if title == "/" {
		title = "fibr"
	}

	var contentURL, description string
	var extraField slack.Text

	switch event.Type {
	case provider.UploadEvent:
		description = "💾 A file has been uploaded"
		contentURL = event.BrowserURL()
	case provider.RenameEvent:
		description = "✏️ An item has been renamed"
		extraField = slack.NewText(fmt.Sprintf("*to*\n%s", event.GetTo()))
		contentURL = event.BrowserURL()
	case provider.DescriptionEvent:
		description = "💬 " + event.Metadata["description"]
		contentURL = event.StoryURL(event.Item.ID)
	}

	section := slack.NewSection(slack.NewText(description)).AddField(slack.NewText(fmt.Sprintf("*item*\n%s", event.GetName())))
	if len(extraField.Text) != 0 {
		section = section.AddField(extraField)
	}

	if s.thumbnail.CanHaveThumbnail(event.Item) {
		section.Accessory = slack.NewAccessory(event.GetURL()+"?thumbnail", fmt.Sprintf("Thumbnail of %s", event.Item.Name()))
	}

	return send(ctx, webhook.ID, request.Post(webhook.URL), slack.NewResponse(s.eventText(event)).AddBlock(slack.NewSection(slack.NewText(fmt.Sprintf("*<%s|%s>*", contentURL, title)))).AddBlock(section))
}

func (s *Service) telegramHandle(ctx context.Context, webhook provider.Webhook, event provider.Event) (int, error) {
	return send(ctx, webhook.ID, request.Post(fmt.Sprintf("%s&text=%s", webhook.URL, url.QueryEscape(s.eventText(event)))), nil)
}

func (s *Service) eventText(event provider.Event) string {
	switch event.Type {
	case provider.AccessEvent:
		return s.accessEvent(event)
	case provider.CreateDir:
		return fmt.Sprintf("🗂 A directory `%s` has been created: %s", event.Item.Name(), event.GetURL())
	case provider.UploadEvent:
		return fmt.Sprintf("💾 A file has been uploaded: %s?browser", event.GetURL())
	case provider.RenameEvent:
		return fmt.Sprintf("✏️ `%s` has been renamed to `%s`: %s?browser", event.Item.Pathname, event.New.Pathname, event.GetURL())
	case provider.DeleteEvent:
		return fmt.Sprintf("❌ `%s` has been deleted : %s", event.Item.Name(), event.GetURL())
	case provider.DescriptionEvent:
		contentURL := event.GetURL()
		return fmt.Sprintf("💬 %s %s", event.Metadata["description"], fmt.Sprintf("%s/?d=story#%s", contentURL[:strings.LastIndex(contentURL, "/")], event.Item.ID))
	case provider.StartEvent:
		return fmt.Sprintf("🚀 Fibr starts routine for path `%s`", event.Item.Pathname)
	default:
		return fmt.Sprintf("🙄 Event `%s` occurred on `%s`", event.Type, event.Item.Name())
	}
}

func (s *Service) accessEvent(event provider.Event) string {
	content := strings.Builder{}
	content.WriteString(fmt.Sprintf("💻 Someone connected to Fibr from %s at %s", s.rendererService.PublicURL(event.URL), event.Time.Format(time.RFC3339)))

	if len(event.Metadata) > 0 {
		content.WriteString("\n```\n")

		for key, value := range event.Metadata {
			content.WriteString(fmt.Sprintf("%s: %s\n", key, value))
		}

		content.WriteString("```")
	}

	return content.String()
}

func (s *Service) deleteItem(ctx context.Context, item absto.Item) error {
	return s.Exclusive(ctx, item.ID, func(_ context.Context) error {
		for id, webhook := range s.webhooks {
			if webhook.Pathname == item.Pathname {
				if err := s.delete(ctx, id); err != nil {
					return fmt.Errorf("delete webhook `%s`: %w", id, err)
				}
			}
		}

		return nil
	})
}
