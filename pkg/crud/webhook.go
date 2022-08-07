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

func (a App) createWebhook(w http.ResponseWriter, r *http.Request, request provider.Request) {
	var err error
	err = r.ParseForm()
	if err != nil {
		a.error(w, r, request, model.WrapInvalid(fmt.Errorf("parse form: %s", err)))
		return
	}

	recursive, kind, webhookURL, eventTypes, err := checkWebhookForm(r)
	if err != nil {
		a.error(w, r, request, err)
		return
	}

	ctx := r.Context()

	info, err := a.storageApp.Info(ctx, request.Path)
	if err != nil {
		if absto.IsNotExist(err) {
			a.error(w, r, request, model.WrapNotFound(err))
		} else {
			a.error(w, r, request, model.WrapInternal(err))
		}
		return
	}

	if !info.IsDir {
		a.error(w, r, request, model.WrapInvalid(errors.New("webhook are only available on directories")))
		return
	}

	id, err := a.webhookApp.Create(ctx, info.Pathname, recursive, kind, webhookURL, eventTypes)
	if err != nil {
		a.error(w, r, request, model.WrapInternal(err))
		return
	}

	if r.Header.Get("Accept") == "text/plain" {
		w.WriteHeader(http.StatusCreated)
		provider.SafeWrite(w, id)

		return
	}

	a.rendererApp.Redirect(w, r, fmt.Sprintf("%s?d=%s#webhook-list", request.Path, request.Display), renderer.NewSuccessMessage("Webhook successfully created with ID: %s", id))
}

func checkWebhookForm(r *http.Request) (recursive bool, kind provider.WebhookKind, webhookURL string, eventTypes []provider.EventType, err error) {
	recursive, err = getFormBool(r.Form.Get("recursive"))
	if err != nil {
		err = model.WrapInvalid(err)
		return
	}

	kind, err = provider.ParseWebhookKind(r.Form.Get("kind"))
	if err != nil {
		err = model.WrapInvalid(fmt.Errorf("parse kind: %s", err))
		return
	}

	webhookURL = r.Form.Get("url")
	if len(webhookURL) == 0 {
		err = model.WrapInvalid(errors.New("url or token is required"))
		return
	}

	if kind == provider.Telegram {
		chatID := r.Form.Get("chat-id")
		if len(chatID) == 0 {
			err = model.WrapInvalid(errors.New("chat ID is required"))
			return
		}

		webhookURL = generateTelegramURL(webhookURL, chatID)
	} else {
		if _, err = url.Parse(webhookURL); err != nil {
			err = model.WrapInvalid(fmt.Errorf("parse url: %s", err))
			return
		}
	}

	rawEventTypes := r.Form["types"]
	if len(rawEventTypes) == 0 {
		err = model.WrapInvalid(errors.New("at least one event type has to be chosen"))
		return
	}

	eventTypes = make([]provider.EventType, len(rawEventTypes))
	for i, rawEventType := range rawEventTypes {
		var eType provider.EventType
		eType, err = provider.ParseEventType(rawEventType)
		if err != nil {
			err = model.WrapInvalid(err)
			return
		}
		eventTypes[i] = eType
	}

	return
}

func (a App) deleteWebhook(w http.ResponseWriter, r *http.Request, request provider.Request) {
	id := r.FormValue("id")

	if err := a.webhookApp.Delete(r.Context(), id); err != nil {
		a.error(w, r, request, model.WrapInternal(err))
		return
	}

	if r.Header.Get("Accept") == "text/plain" {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	a.rendererApp.Redirect(w, r, fmt.Sprintf("%s?d=%s#webhook-list", request.Path, request.Display), renderer.NewSuccessMessage("Webhook with id %s successfully deleted", id))
}
