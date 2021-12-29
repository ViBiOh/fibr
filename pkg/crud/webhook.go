package crud

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"

	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/httputils/v4/pkg/model"
	"github.com/ViBiOh/httputils/v4/pkg/renderer"
)

func generateTelegramURL(botToken, chatID string) string {
	return fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage?chat_id=%s", url.PathEscape(botToken), url.QueryEscape(chatID))
}

func (a App) createWebhook(w http.ResponseWriter, r *http.Request, request provider.Request) {
	if !request.CanWebhook {
		a.rendererApp.Error(w, r, model.WrapForbidden(ErrNotAuthorized))
		return
	}

	var err error
	err = r.ParseForm()
	if err != nil {
		a.rendererApp.Error(w, r, model.WrapInvalid(fmt.Errorf("unable to parse form: %s", err)))
		return
	}

	recursive, kind, webhookURL, eventTypes, err := checkWebhookForm(r)
	if err != nil {
		a.rendererApp.Error(w, r, err)
		return
	}

	info, err := a.storageApp.Info(request.Path)
	if err != nil {
		if provider.IsNotExist(err) {
			a.rendererApp.Error(w, r, model.WrapNotFound(err))
		} else {
			a.rendererApp.Error(w, r, model.WrapInternal(err))
		}
		return
	}

	if !info.IsDir {
		a.rendererApp.Error(w, r, model.WrapInvalid(errors.New("webhook are only available on directories")))
		return
	}

	id, err := a.webhookApp.Create(info.Pathname, recursive, kind, webhookURL, eventTypes)
	if err != nil {
		a.rendererApp.Error(w, r, model.WrapInternal(err))
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
		err = model.WrapInvalid(fmt.Errorf("unable to parse kind: %s", err))
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
			err = model.WrapInvalid(fmt.Errorf("unable to parse url: %s", err))
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
	if !request.CanWebhook {
		a.rendererApp.Error(w, r, model.WrapForbidden(ErrNotAuthorized))
		return
	}

	id := r.FormValue("id")

	if err := a.webhookApp.Delete(id); err != nil {
		a.rendererApp.Error(w, r, model.WrapInternal(err))
		return
	}

	if r.Header.Get("Accept") == "text/plain" {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	a.rendererApp.Redirect(w, r, fmt.Sprintf("%s?d=%s#webhook-list", request.Path, request.Display), renderer.NewSuccessMessage("Webhook with id %s successfully deleted", id))
}
