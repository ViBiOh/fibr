package fibr

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/ViBiOh/auth/v2/pkg/auth"
	"github.com/ViBiOh/auth/v2/pkg/ident"
	authModel "github.com/ViBiOh/auth/v2/pkg/model"
	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/httputils/v4/pkg/logger"
	"github.com/ViBiOh/httputils/v4/pkg/model"
	"github.com/ViBiOh/httputils/v4/pkg/renderer"
)

// App of package
type App struct {
	loginApp    provider.Auth
	crudApp     provider.Crud
	shareApp    provider.ShareManager
	webhookApp  provider.WebhookManager
	rendererApp renderer.App
}

// New creates new App from Config
func New(crudApp provider.Crud, rendererApp renderer.App, shareApp provider.ShareManager, webhookApp provider.WebhookManager, loginApp provider.Auth) App {
	return App{
		crudApp:     crudApp,
		rendererApp: rendererApp,
		shareApp:    shareApp,
		webhookApp:  webhookApp,
		loginApp:    loginApp,
	}
}

func (a App) parseShare(request *provider.Request, authorizationHeader string) error {
	share := a.shareApp.Get(request.Path)
	if len(share.ID) == 0 {
		return nil
	}

	if err := share.CheckPassword(authorizationHeader); err != nil {
		return err
	}

	request.Share = share
	request.CanEdit = share.Edit
	request.Path = strings.TrimPrefix(request.Path, fmt.Sprintf("/%s", share.ID))

	return nil
}

func convertAuthenticationError(err error) error {
	if errors.Is(err, auth.ErrForbidden) {
		return model.WrapForbidden(errors.New("you're not authorized to speak to me"))
	}

	if errors.Is(err, ident.ErrMalformedAuth) {
		return model.WrapInvalid(err)
	}

	return model.WrapUnauthorized(err)
}

func parsePreferences(r *http.Request) provider.Preferences {
	var preferences provider.Preferences

	if cookie, err := r.Cookie("list_layout_paths"); err == nil {
		if value := cookie.Value; len(value) > 0 {
			preferences.ListLayoutPath = strings.Split(value, ",")
		}
	}

	return preferences
}

func parseDisplay(r *http.Request) string {
	display := r.URL.Query().Get("d")

	if len(display) != 0 {
		return display
	}

	return provider.DefaultDisplay
}

func (a App) parseRequest(r *http.Request) (provider.Request, error) {
	request := provider.Request{
		Path:        r.URL.Path,
		CanEdit:     false,
		CanShare:    false,
		Display:     parseDisplay(r),
		Preferences: parsePreferences(r),
	}

	if err := a.parseShare(&request, r.Header.Get("Authorization")); err != nil {
		logRequest(r)
		return request, model.WrapUnauthorized(err)
	}

	if len(request.Share.ID) != 0 {
		if request.Share.IsExpired(time.Now()) {
			return request, model.WrapNotFound(errors.New("link has expired"))
		}

		return request, nil
	}

	if a.loginApp == nil {
		request.CanEdit = true
		request.CanShare = a.shareApp.Enabled()
		request.CanWebhook = a.webhookApp.Enabled()
		return request, nil
	}

	_, user, err := a.loginApp.IsAuthenticated(r)
	if err != nil {
		logRequest(r)
		return request, convertAuthenticationError(err)
	}

	if a.loginApp.IsAuthorized(authModel.StoreUser(r.Context(), user), "admin") {
		request.CanEdit = true
		request.CanShare = a.shareApp.Enabled()
		request.CanWebhook = a.webhookApp.Enabled()
	}

	return request, nil
}

func logRequest(r *http.Request) {
	ip := r.Header.Get("X-Forwarded-For")
	if len(ip) == 0 {
		ip = r.Header.Get("X-Real-Ip")
	}
	if len(ip) == 0 {
		ip = r.RemoteAddr
	}

	logger.Warn("Unauthenticated request: %s %s from %s", r.Method, r.URL.String(), ip)
}
