package fibr

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"path"
	"strings"
	"time"

	"github.com/ViBiOh/auth/v2/pkg/auth"
	"github.com/ViBiOh/auth/v2/pkg/ident"
	authModel "github.com/ViBiOh/auth/v2/pkg/model"
	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/httputils/v4/pkg/model"
	"github.com/ViBiOh/httputils/v4/pkg/renderer"
)

type Service struct {
	login    provider.Auth
	crud     provider.Crud
	share    provider.ShareManager
	webhook  provider.WebhookManager
	renderer *renderer.Service
}

func New(crudService provider.Crud, rendererService *renderer.Service, shareService provider.ShareManager, webhookService provider.WebhookManager, loginService provider.Auth) Service {
	return Service{
		crud:     crudService,
		renderer: rendererService,
		share:    shareService,
		webhook:  webhookService,
		login:    loginService,
	}
}

func (s Service) parseRequest(r *http.Request) (provider.Request, error) {
	request := provider.Request{
		Path:        r.URL.Path,
		CanEdit:     false,
		CanShare:    false,
		Preferences: parsePreferences(r),
	}

	if !strings.HasSuffix(request.Path, "/") {
		request.Item = path.Base(request.Path)
		request.Path = path.Dir(request.Path)
	}

	if !strings.HasPrefix(request.Path, "/") {
		request.Path = "/" + request.Path
	}

	if err := s.parseShare(r.Context(), &request, r.Header.Get("Authorization")); err != nil {
		return request, model.WrapUnauthorized(err)
	}

	if len(request.Display) == 0 {
		if displayParam := r.URL.Query().Get("d"); len(displayParam) == 0 {
			request.Display = request.LayoutPath(request.AbsoluteURL(""))
		} else {
			request.Display = provider.ParseDisplay(displayParam)
		}
	}

	request = request.UpdatePreferences()

	if !request.Share.IsZero() {
		if request.Share.IsExpired(time.Now()) {
			return request, model.WrapNotFound(errors.New("link has expired"))
		}

		return request, nil
	}

	if s.login == nil {
		request.CanEdit = true
		request.CanShare = true
		request.CanWebhook = true

		return request, nil
	}

	_, user, err := s.login.IsAuthenticated(r)
	if err != nil {
		return request, convertAuthenticationError(err)
	}

	if s.login.IsAuthorized(authModel.StoreUser(r.Context(), user), "admin") {
		request.CanEdit = true
		request.CanShare = true
		request.CanWebhook = true
	}

	return request, nil
}

func parsePreferences(r *http.Request) provider.Preferences {
	var cookieValue string

	if cookie, err := r.Cookie(provider.LayoutPathsCookieName); err == nil {
		cookieValue = cookie.Value
	}

	return provider.ParsePreferences(cookieValue)
}

func (s Service) parseShare(ctx context.Context, request *provider.Request, authorizationHeader string) error {
	share := s.share.Get(request.Filepath())
	if share.IsZero() {
		return nil
	}

	if err := share.CheckPassword(ctx, authorizationHeader, s.share); err != nil {
		return err
	}

	request.Share = share
	request.CanEdit = share.Edit
	request.Path = strings.TrimPrefix(request.Path, fmt.Sprintf("/%s", share.ID))

	if share.Story {
		request.Display = provider.StoryDisplay
	}

	if share.File {
		request.Path = ""
	}

	return nil
}

func convertAuthenticationError(err error) error {
	if errors.Is(err, auth.ErrForbidden) {
		return model.WrapForbidden(errors.New("you're not authorized to speak to me with this terms"))
	}

	if errors.Is(err, ident.ErrMalformedAuth) {
		return model.WrapInvalid(err)
	}

	return model.WrapUnauthorized(err)
}
