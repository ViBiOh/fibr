package fibr

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/ViBiOh/auth/v2/pkg/auth"
	"github.com/ViBiOh/auth/v2/pkg/ident"
	authMiddleware "github.com/ViBiOh/auth/v2/pkg/middleware"
	"github.com/ViBiOh/fibr/pkg/crud"
	"github.com/ViBiOh/fibr/pkg/metadata"
	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/fibr/pkg/renderer"
	"github.com/ViBiOh/httputils/v4/pkg/httperror"
	"github.com/ViBiOh/httputils/v4/pkg/query"
)

// App of package
type App interface {
	Handler() http.Handler
}

type app struct {
	loginApp    authMiddleware.App
	crudApp     crud.App
	rendererApp renderer.App
	metadataApp metadata.App
}

// New creates new App from Config
func New(crudApp crud.App, rendererApp renderer.App, metadataApp metadata.App, loginApp authMiddleware.App) App {
	return &app{
		crudApp:     crudApp,
		rendererApp: rendererApp,
		loginApp:    loginApp,
		metadataApp: metadataApp,
	}
}

func (a app) parseShare(request *provider.Request, authorizationHeader string) error {
	share := a.metadataApp.GetShare(request.Path)
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

func convertAuthenticationError(err error) *provider.Error {
	if errors.Is(err, auth.ErrForbidden) {
		return provider.NewError(http.StatusForbidden, errors.New("you're not authorized to speak to me"))
	}

	if errors.Is(err, ident.ErrMalformedAuth) {
		return provider.NewError(http.StatusBadRequest, err)
	}

	return provider.NewError(http.StatusUnauthorized, err)
}

func (a app) parseRequest(r *http.Request) (provider.Request, *provider.Error) {
	preferences := provider.Preferences{}
	if cookie, err := r.Cookie("list_layout_paths"); err == nil {
		if value := cookie.Value; len(value) > 0 {
			preferences.ListLayoutPath = strings.Split(value, ",")
		}
	}

	request := provider.Request{
		Path:        r.URL.Path,
		CanEdit:     false,
		CanShare:    false,
		Display:     r.URL.Query().Get("d"),
		Preferences: preferences,
	}

	if err := a.parseShare(&request, r.Header.Get("Authorization")); err != nil {
		return request, provider.NewError(http.StatusUnauthorized, err)
	}

	if len(request.Share.ID) != 0 {
		if request.Share.IsExpired(time.Now()) {
			return request, provider.NewError(http.StatusNotFound, errors.New("link has expired"))
		}

		return request, nil
	}

	if a.loginApp == nil {
		request.CanEdit = true
		request.CanShare = a.metadataApp.Enabled()
		return request, nil
	}

	_, user, err := a.loginApp.IsAuthenticated(r, "")
	if err != nil {
		return request, convertAuthenticationError(err)
	}

	if a.loginApp.HasProfile(r.Context(), user, "admin") {
		request.CanEdit = true
		request.CanShare = a.metadataApp.Enabled()
	}

	return request, nil
}

func (a app) handleRequest(w http.ResponseWriter, r *http.Request, request provider.Request) {
	switch r.Method {
	case http.MethodGet:
		a.crudApp.Get(w, r, request)
	case http.MethodPost:
		a.crudApp.Post(w, r, request)
	case http.MethodPut:
		a.crudApp.Create(w, r, request)
	case http.MethodPatch:
		a.crudApp.Rename(w, r, request)
	case http.MethodDelete:
		a.crudApp.Delete(w, r, request)
	default:
		httperror.NotFound(w)
	}
}

// Handler for request. Should be use with net/http
func (a app) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !isMethodAllowed(r) {
			a.rendererApp.Error(w, provider.Request{}, provider.NewError(http.StatusMethodNotAllowed, errors.New("you lack of method for calling me")))
			return
		}

		if a.crudApp.ServeStatic(w, r) {
			return
		}

		if query.GetBool(r, "redirect") {
			http.Redirect(w, r, r.URL.Path, http.StatusFound)
			return
		}

		request, err := a.parseRequest(r)
		if err != nil {
			a.rendererApp.Error(w, request, err)
			return
		}

		a.handleRequest(w, r, request)
	})
}
