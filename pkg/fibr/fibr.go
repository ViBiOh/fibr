package fibr

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/ViBiOh/auth/v2/pkg/auth"
	login "github.com/ViBiOh/auth/v2/pkg/handler"
	"github.com/ViBiOh/auth/v2/pkg/ident"
	"github.com/ViBiOh/fibr/pkg/crud"
	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/fibr/pkg/renderer"
	"github.com/ViBiOh/httputils/v3/pkg/httperror"
	"github.com/ViBiOh/httputils/v3/pkg/query"
)

// App of package
type App interface {
	Handler() http.Handler
}

type app struct {
	login    login.App
	crud     crud.App
	renderer renderer.App
}

// New creates new App from Config
func New(crudApp crud.App, rendererApp renderer.App, loginApp login.App) App {
	return &app{
		crud:     crudApp,
		renderer: rendererApp,
		login:    loginApp,
	}
}

func (a app) parseShare(r *http.Request, request *provider.Request) error {
	share := a.crud.GetShare(request.Path)
	if share == nil {
		return nil
	}

	if err := share.CheckPassword(r); err != nil {
		return err
	}

	request.Share = share
	request.CanEdit = share.Edit
	request.Path = strings.TrimPrefix(request.Path, fmt.Sprintf("/%s", share.ID))

	return nil
}

func (a app) handleAnonymousRequest(r *http.Request, err error) *provider.Error {
	if auth.ErrForbidden == err {
		return provider.NewError(http.StatusForbidden, errors.New("you're not authorized to speak to me"))
	}

	if err == ident.ErrMalformedAuth {
		return provider.NewError(http.StatusBadRequest, err)
	}

	return provider.NewError(http.StatusUnauthorized, err)
}

func (a app) parseRequest(r *http.Request) (provider.Request, *provider.Error) {
	request := provider.Request{
		Path:     r.URL.Path,
		CanEdit:  false,
		CanShare: false,
		Display:  r.URL.Query().Get("d"),
	}

	if err := a.parseShare(r, &request); err != nil {
		return request, provider.NewError(http.StatusUnauthorized, err)
	}

	if request.Share != nil {
		return request, nil
	}

	_, user, err := a.login.IsAuthenticated(r, "")
	if err != nil {
		return request, a.handleAnonymousRequest(r, err)
	}

	if a.login.HasProfile(user, "admin") {
		request.CanEdit = true
		request.CanShare = true
	}

	return request, nil
}

func (a app) handleRequest(w http.ResponseWriter, r *http.Request, request provider.Request) {
	switch r.Method {
	case http.MethodGet:
		a.crud.Get(w, r, request)
	case http.MethodPost:
		a.crud.Post(w, r, request)
	case http.MethodPut:
		a.crud.Create(w, r, request)
	case http.MethodPatch:
		a.crud.Rename(w, r, request)
	case http.MethodDelete:
		a.crud.Delete(w, r, request)
	default:
		httperror.NotFound(w)
	}
}

// Handler for request. Should be use with net/http
func (a app) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !isMethodAllowed(r) {
			a.renderer.Error(w, provider.Request{}, provider.NewError(http.StatusMethodNotAllowed, errors.New("you lack of method for calling me")))
			return
		}

		if a.crud.ServeStatic(w, r) {
			return
		}

		if query.GetBool(r, "redirect") {
			http.Redirect(w, r, r.URL.Path, http.StatusFound)
			return
		}

		request, err := a.parseRequest(r)
		if err != nil {
			a.renderer.Error(w, request, err)
			return
		}

		a.handleRequest(w, r, request)
	})
}
