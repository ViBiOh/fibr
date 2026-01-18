package fibr

import (
	"context"
	"errors"
	"net/http"
	"path"
	"strings"
	"time"

	authModel "github.com/ViBiOh/auth/v3/pkg/model"
	"github.com/ViBiOh/fibr/pkg/cookie"
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
	cookie   cookie.Service
}

func New(crud provider.Crud, renderer *renderer.Service, share provider.ShareManager, webhook provider.WebhookManager, login provider.Auth, cookie cookie.Service) Service {
	return Service{
		crud:     crud,
		renderer: renderer,
		share:    share,
		webhook:  webhook,
		login:    login,
		cookie:   cookie,
	}
}

func (s Service) parseRequest(r *http.Request) (provider.Request, error) {
	ctx := r.Context()

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

	login, password, basicOK := r.BasicAuth()

	if err := s.parseShare(ctx, &request, password); err != nil {
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

	if !basicOK {
		return request, convertAuthenticationError(authModel.ErrMalformedContent)
	}

	user, err := s.login.GetBasicUser(ctx, login, password)
	if err != nil {
		return request, convertAuthenticationError(err)
	}

	if s.login.IsAuthorized(ctx, user, "admin") {
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

func (s Service) parseShare(ctx context.Context, request *provider.Request, password string) error {
	share := s.share.Get(request.Filepath())
	if share.IsZero() {
		return nil
	}

	if err := share.CheckPassword(ctx, password, s.share); err != nil {
		return err
	}

	request.Share = share
	request.CanEdit = share.Edit
	request.Path = strings.TrimPrefix(request.Path, "/"+share.ID)

	if share.Story {
		request.Display = provider.StoryDisplay
	}

	if share.File {
		request.Path = ""
	}

	if request.Path == "/" && request.Item == share.ID {
		request.Item = ""
	}

	return nil
}

func convertAuthenticationError(err error) error {
	if errors.Is(err, authModel.ErrForbidden) {
		return model.WrapForbidden(errors.New("you're not authorized to speak to me with this terms"))
	}

	if errors.Is(err, authModel.ErrMalformedContent) || errors.Is(err, authModel.ErrInvalidCredentials) {
		return model.WrapUnauthorized(err)
	}

	return model.WrapInvalid(err)
}
