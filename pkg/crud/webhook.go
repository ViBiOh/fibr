package crud

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"

	absto "github.com/ViBiOh/absto/pkg/model"
	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/httputils/v4/pkg/model"
	"github.com/ViBiOh/httputils/v4/pkg/renderer"
)

func generateTelegramURL(botToken, chatID string) string {
	return fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage?chat_id=%s", url.PathEscape(botToken), url.QueryEscape(chatID))
}

func (s *Service) createWebhook(w http.ResponseWriter, r *http.Request, request provider.Request) {
	var err error
	err = r.ParseForm()
	if err != nil {
		s.error(w, r, request, model.WrapInvalid(fmt.Errorf("parse form: %w", err)))
		return
	}

	recursive, kind, webhookURL, eventTypes, err := checkWebhookForm(r)
	if err != nil {
		s.error(w, r, request, err)
		return
	}

	if !request.CanWebhook && kind != provider.Push {
		s.error(w, r, request, model.WrapForbidden(ErrNotAuthorized))
		return
	}

	ctx := r.Context()

	info, err := s.storage.Stat(ctx, request.Filepath())
	if err != nil {
		if absto.IsNotExist(err) {
			s.error(w, r, request, model.WrapNotFound(err))
		} else {
			s.error(w, r, request, model.WrapInternal(err))
		}

		return
	}

	if !info.IsDir() {
		s.error(w, r, request, model.WrapInvalid(errors.New("webhook are only available on directories")))
		return
	}

	id, err := s.webhook.Create(ctx, info.Pathname, recursive, kind, webhookURL, eventTypes)
	if err != nil {
		s.error(w, r, request, model.WrapInternal(err))
		return
	}

	if r.Header.Get("Accept") == "text/plain" {
		w.WriteHeader(http.StatusCreated)
		provider.SafeWrite(ctx, w, id)

		return
	}

	if kind == provider.Push {
		s.renderer.Redirect(w, r, fmt.Sprintf("%s?d=%s", request.AbsoluteURL(""), request.Display), renderer.NewSuccessMessage("Push notification registered!"))
		return
	}

	s.renderer.Redirect(w, r, fmt.Sprintf("%s?d=%s#webhook-list", request.AbsoluteURL(""), request.Display), renderer.NewSuccessMessage("Webhook successfully created with ID: %s", id))
}

func checkWebhookForm(r *http.Request) (recursive bool, kind provider.WebhookKind, webhookURL string, eventTypes []provider.EventType, err error) {
	recursive, err = getFormBool(r.Form.Get("recursive"))
	if err != nil {
		err = model.WrapInvalid(err)
		return recursive, kind, webhookURL, eventTypes, err
	}

	kind, err = provider.ParseWebhookKind(r.Form.Get("kind"))
	if err != nil {
		err = model.WrapInvalid(fmt.Errorf("parse kind: %w", err))
		return recursive, kind, webhookURL, eventTypes, err
	}

	webhookURL = r.Form.Get("url")
	if len(webhookURL) == 0 {
		err = model.WrapInvalid(errors.New("url or token is required"))
		return recursive, kind, webhookURL, eventTypes, err
	}

	if kind == provider.Telegram {
		chatID := r.Form.Get("chat-id")
		if len(chatID) == 0 {
			err = model.WrapInvalid(errors.New("chat ID is required"))
			return recursive, kind, webhookURL, eventTypes, err
		}

		webhookURL = generateTelegramURL(webhookURL, chatID)
	} else {
		if _, err = url.Parse(webhookURL); err != nil {
			err = model.WrapInvalid(fmt.Errorf("parse url: %w", err))
			return recursive, kind, webhookURL, eventTypes, err
		}
	}

	rawEventTypes := r.Form["types"]
	if len(rawEventTypes) == 0 {
		err = model.WrapInvalid(errors.New("at least one event type has to be chosen"))
		return recursive, kind, webhookURL, eventTypes, err
	}

	eventTypes = make([]provider.EventType, len(rawEventTypes))
	for i, rawEventType := range rawEventTypes {
		var eType provider.EventType
		eType, err = provider.ParseEventType(rawEventType)
		if err != nil {
			err = model.WrapInvalid(err)
			return recursive, kind, webhookURL, eventTypes, err
		}
		eventTypes[i] = eType
	}

	return recursive, kind, webhookURL, eventTypes, err
}

func (s *Service) deleteWebhook(w http.ResponseWriter, r *http.Request, request provider.Request) {
	webhook := s.webhook.Get(r.FormValue("id"))

	if !request.CanWebhook && webhook.Kind != provider.Push {
		s.error(w, r, request, model.WrapForbidden(ErrNotAuthorized))
		return
	}

	if len(webhook.ID) == 0 {
		s.error(w, r, request, model.WrapNotFound(errors.New("webhook not found")))
		return
	}

	ctx := r.Context()

	if err := s.webhook.Delete(ctx, webhook.ID); err != nil {
		s.error(w, r, request, model.WrapInternal(err))
		return
	}

	if r.Header.Get("Accept") == "text/plain" {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	if webhook.Kind == provider.Push {
		s.renderer.Redirect(w, r, fmt.Sprintf("%s?d=%s", request.AbsoluteURL(""), request.Display), renderer.NewSuccessMessage("Push notification removed!"))
		return
	}

	s.renderer.Redirect(w, r, fmt.Sprintf("%s?d=%s#webhook-list", request.AbsoluteURL(""), request.Display), renderer.NewSuccessMessage("Webhook with id %s successfully deleted", webhook.ID))
}
