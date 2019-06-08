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
	"github.com/ViBiOh/httputils/pkg/httperror"
	"github.com/pkg/errors"
)

var (
	errEmptyAuthorizationHeader = errors.New("empty authorization header")
)

// App of package
type App interface {
	Handler() http.Handler
}

type app struct {
	crud     *crud.App
	renderer *renderer.App
	auth     auth.App
}

// New creates new App from Config
func New(crudApp *crud.App, rendererApp *renderer.App, authApp auth.App) App {
	return &app{
		crud:     crudApp,
		renderer: rendererApp,
		auth:     authApp,
	}
}

func (a app) checkShare(w http.ResponseWriter, r *http.Request, request *provider.Request) error {
	if share := a.crud.GetShare(request.Path); share != nil {
		request.Share = share
		request.CanEdit = share.Edit
		request.Path = strings.TrimPrefix(request.Path, fmt.Sprintf("/%s", share.ID))

		if share.Password != "" {
			if err := checkSharePassword(r, share); err != nil {
				w.Header().Add("WWW-Authenticate", "Basic realm=\"Password required\" charset=\"UTF-8\"")
				return err
			}
		}
	}

	return nil
}

func (a app) handleAnonymousRequest(w http.ResponseWriter, r *http.Request, err error) {
	if auth.ErrForbidden == err {
		a.renderer.Error(w, http.StatusForbidden, errors.New("you're not authorized to speak to me"))
		return
	}

	if err == ident.ErrMalformedAuth || err == ident.ErrUnknownIdentType {
		a.renderer.Error(w, http.StatusBadRequest, err)
		return
	}

	w.Header().Add("WWW-Authenticate", "Basic charset=\"UTF-8\"")
	a.renderer.Error(w, http.StatusUnauthorized, err)
}

func (a app) checkAllowedMethod(r *http.Request) bool {
	return r.Method == http.MethodGet || r.Method == http.MethodPost || r.Method == http.MethodPut || r.Method == http.MethodPatch || r.Method == http.MethodDelete
}

func (a app) checkRequest(w http.ResponseWriter, r *http.Request) *provider.Request {
	request := &provider.Request{
		Path:     r.URL.Path,
		CanEdit:  false,
		CanShare: false,
	}

	if err := a.checkShare(w, r, request); err != nil {
		a.renderer.Error(w, http.StatusUnauthorized, err)
		return nil
	}

	if request.Share == nil {
		user, err := a.auth.IsAuthenticated(r)
		if err != nil {
			a.handleAnonymousRequest(w, r, err)
			return nil
		}

		if user != nil && user.HasProfile("admin") {
			request.CanEdit = true
			request.CanShare = true
		}
	}

	return request
}

func (a app) handleRequest(w http.ResponseWriter, r *http.Request, config *provider.Request) {
	switch r.Method {
	case http.MethodGet:
		a.crud.Get(w, r, config)
	case http.MethodPost:
		a.crud.Post(w, r, config)
	case http.MethodPut:
		a.crud.Create(w, r, config)
	case http.MethodPatch:
		a.crud.Rename(w, r, config)
	case http.MethodDelete:
		a.crud.Delete(w, r, config)
	default:
		httperror.NotFound(w)
	}
}

// Handler for request. Should be use with net/http
func (a app) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		if !a.checkAllowedMethod(r) {
			a.renderer.Error(w, http.StatusMethodNotAllowed, errors.New("you lack of accurate method for calling me"))
			return
		}

		if strings.Contains(r.URL.Path, "..") {
			a.renderer.Error(w, http.StatusForbidden, errors.New("you can't speak to my parent"))
			return
		}

		if a.crud.CheckAndServeSEO(w, r) {
			return
		}

		request := a.checkRequest(w, r)
		if request != nil {
			a.handleRequest(w, r, request)
		}
	})
}
