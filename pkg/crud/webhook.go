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

func (a App) createWebhook(w http.ResponseWriter, r *http.Request, request provider.Request) {
	if !a.webhookApp.Enabled() {
		a.rendererApp.Error(w, r, model.WrapInternal(errors.New("webhook is disabled")))
		return
	}

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

	recursive, err := getFormBool(r.Form.Get("recursive"))
	if err != nil {
		a.rendererApp.Error(w, r, model.WrapInvalid(err))
		return
	}

	target, err := url.Parse(r.Form.Get("url"))
	if err != nil {
		a.rendererApp.Error(w, r, model.WrapInvalid(fmt.Errorf("unable to parse url: %s", err)))
		return
	}

	rawEventTypes := r.Form["types"]
	eventTypes := make([]provider.EventType, len(rawEventTypes))
	for i, rawEventType := range rawEventTypes {
		eType, err := provider.ParseEventType(rawEventType)
		if err != nil {
			a.rendererApp.Error(w, r, model.WrapInvalid(err))
			return
		}
		eventTypes[i] = eType
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

	id, err := a.webhookApp.Create(info.Pathname, recursive, target.String(), eventTypes)
	if err != nil {
		a.rendererApp.Error(w, r, model.WrapInternal(err))
		return
	}

	if r.Header.Get("Accept") == "text/plain" {
		w.WriteHeader(http.StatusCreated)
		provider.SafeWrite(w, id)

		return
	}

	a.rendererApp.Redirect(w, r, fmt.Sprintf("%s/?d=%s#webhook-list", request.URL(""), request.Layout("")), renderer.NewSuccessMessage("Webhook successfully created with ID: %s", id))
}

func (a App) deleteWebhook(w http.ResponseWriter, r *http.Request, request provider.Request) {
	if !a.webhookApp.Enabled() {
		a.rendererApp.Error(w, r, model.WrapInternal(errors.New("webhook is disabled")))
		return
	}

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

	a.rendererApp.Redirect(w, r, fmt.Sprintf("%s/?d=%s#webhook-list", request.URL(""), request.Layout("")), renderer.NewSuccessMessage("Webhook with id %s successfully deleted", id))
}
