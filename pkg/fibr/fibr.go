package fibr

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/ViBiOh/auth/pkg/auth"
	"github.com/ViBiOh/auth/pkg/ident"
	"github.com/ViBiOh/fibr/pkg/crud"
	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/fibr/pkg/renderer"
	"github.com/ViBiOh/httputils/pkg/errors"
	"github.com/ViBiOh/httputils/pkg/httperror"
)

var (
	errEmptyAuthorizationHeader = errors.New("empty authorization header")
)

// App of package
type App interface {
	Handler() http.Handler
}

type app struct {
	auth     auth.App
	crud     crud.App
	renderer renderer.App
}

// New creates new App from Config
func New(crudApp crud.App, rendererApp renderer.App, authApp auth.App) App {
	return &app{
		crud:     crudApp,
		renderer: rendererApp,
		auth:     authApp,
	}
}

func (a app) parseShare(r *http.Request, request *provider.Request) error {
	if share := a.crud.GetShare(request.Path); share != nil {
		request.Share = share
		request.CanEdit = share.Edit
		request.Path = strings.TrimPrefix(request.Path, fmt.Sprintf("/%s", share.ID))

		if err := checkSharePassword(r, share); err != nil {
			return err
		}
	}

	return nil
}

func (a app) handleAnonymousRequest(r *http.Request, err error) *provider.Error {
	if auth.ErrForbidden == err {
		return provider.NewError(http.StatusForbidden, errors.New("you're not authorized to speak to me"))
	}

	if err == ident.ErrMalformedAuth || err == ident.ErrUnknownIdentType {
		return provider.NewError(http.StatusBadRequest, err)
	}

	return provider.NewError(http.StatusUnauthorized, err)
}

func (a app) parseRequest(r *http.Request) (*provider.Request, *provider.Error) {
	request := &provider.Request{
		Path:     r.URL.Path,
		CanEdit:  false,
		CanShare: false,
	}

	if err := a.parseShare(r, request); err != nil {
		return request, provider.NewError(http.StatusUnauthorized, err)
	}

	if request.Share != nil {
		return request, nil
	}

	user, err := a.auth.IsAuthenticated(r)
	if err != nil {
		return request, a.handleAnonymousRequest(r, err)
	}

	if user != nil && user.HasProfile("admin") {
		request.CanEdit = true
		request.CanShare = true
	}

	return request, nil
}

func (a app) handleRequest(w http.ResponseWriter, r *http.Request, request *provider.Request) {
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
			a.renderer.Error(w, provider.NewError(http.StatusMethodNotAllowed, errors.New("you lack of accurate method for calling me")))
			return
		}

		if strings.Contains(r.URL.Path, "..") {
			a.renderer.Error(w, provider.NewError(http.StatusForbidden, errors.New("you can't speak to my parent")))
			return
		}

		if a.crud.CheckAndServeSEO(w, r) {
			return
		}

		if request, err := a.parseRequest(r); err != nil {
			a.renderer.Error(w, err)
		} else if request != nil {
			a.handleRequest(w, r, request)
		}
	})
}
