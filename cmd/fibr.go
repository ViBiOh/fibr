package main

import (
	"encoding/base64"
	"flag"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/ViBiOh/auth/pkg/auth"
	"github.com/ViBiOh/auth/pkg/ident"
	"github.com/ViBiOh/auth/pkg/ident/basic"
	authService "github.com/ViBiOh/auth/pkg/ident/service"
	"github.com/ViBiOh/fibr/pkg/crud"
	"github.com/ViBiOh/fibr/pkg/filesystem"
	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/fibr/pkg/renderer"
	"github.com/ViBiOh/fibr/pkg/thumbnail"
	httputils "github.com/ViBiOh/httputils/pkg"
	"github.com/ViBiOh/httputils/pkg/alcotest"
	"github.com/ViBiOh/httputils/pkg/errors"
	"github.com/ViBiOh/httputils/pkg/gzip"
	"github.com/ViBiOh/httputils/pkg/healthcheck"
	"github.com/ViBiOh/httputils/pkg/httperror"
	"github.com/ViBiOh/httputils/pkg/logger"
	"github.com/ViBiOh/httputils/pkg/opentracing"
	"github.com/ViBiOh/httputils/pkg/owasp"
	"github.com/ViBiOh/httputils/pkg/prometheus"
	"github.com/ViBiOh/httputils/pkg/server"
	"golang.org/x/crypto/bcrypt"
)

var errEmptyAuthorizationHeader = errors.New("empty authorization header")

func checkSharePassword(r *http.Request, share *provider.Share) error {
	header := r.Header.Get("Authorization")
	if header == "" {
		return errEmptyAuthorizationHeader
	}

	data, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(header, "Basic "))
	if err != nil {
		return errors.WithStack(err)
	}

	dataStr := string(data)

	sepIndex := strings.Index(dataStr, ":")
	if sepIndex < 0 {
		return errors.New("invalid format for basic auth")
	}

	password := dataStr[sepIndex+1:]
	if err := bcrypt.CompareHashAndPassword([]byte(share.Password), []byte(password)); err != nil {
		return errors.New("invalid credentials")
	}

	return nil
}

func checkShare(w http.ResponseWriter, r *http.Request, crudApp *crud.App, request *provider.Request) error {
	if share := crudApp.GetShare(request.Path); share != nil {
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

func handleAnonymousRequest(w http.ResponseWriter, r *http.Request, err error, crudApp *crud.App, rendererApp *renderer.App) {
	if auth.ErrForbidden == err {
		rendererApp.Error(w, http.StatusForbidden, errors.New("you're not authorized to speak to me"))
		return
	}

	if err == ident.ErrMalformedAuth || err == ident.ErrUnknownIdentType {
		rendererApp.Error(w, http.StatusBadRequest, err)
		return
	}

	w.Header().Add("WWW-Authenticate", "Basic charset=\"UTF-8\"")
	rendererApp.Error(w, http.StatusUnauthorized, err)
}

func checkAllowedMethod(r *http.Request) bool {
	return r.Method == http.MethodGet || r.Method == http.MethodPost || r.Method == http.MethodPut || r.Method == http.MethodPatch || r.Method == http.MethodDelete
}

func checkRequest(crudApp *crud.App, rendererApp *renderer.App, authApp *auth.App, w http.ResponseWriter, r *http.Request) *provider.Request {
	request := &provider.Request{
		Path:     r.URL.Path,
		CanEdit:  false,
		CanShare: false,
	}

	if err := checkShare(w, r, crudApp, request); err != nil {
		rendererApp.Error(w, http.StatusUnauthorized, err)
		return nil
	}

	if request.Share == nil {
		user, err := authApp.IsAuthenticated(r)
		if err != nil {
			handleAnonymousRequest(w, r, err, crudApp, rendererApp)
			return nil
		}

		if user != nil && user.HasProfile("admin") {
			request.CanEdit = true
			request.CanShare = true
		}
	}

	return request
}

func handleRequest(w http.ResponseWriter, r *http.Request, crudApp *crud.App, config *provider.Request) {
	switch r.Method {
	case http.MethodGet:
		crudApp.Get(w, r, config)
	case http.MethodPost:
		crudApp.Post(w, r, config)
	case http.MethodPut:
		crudApp.Create(w, r, config)
	case http.MethodPatch:
		crudApp.Rename(w, r, config)
	case http.MethodDelete:
		crudApp.Delete(w, r, config)
	default:
		httperror.NotFound(w)
	}
}

func browserHandler(crudApp *crud.App, rendererApp *renderer.App, authApp *auth.App) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !checkAllowedMethod(r) {
			rendererApp.Error(w, http.StatusMethodNotAllowed, errors.New("you lack of accurate method for calling me"))
			return
		}

		if strings.Contains(r.URL.Path, "..") {
			rendererApp.Error(w, http.StatusForbidden, errors.New("you can't speak to my parent"))
			return
		}

		if crudApp.CheckAndServeSEO(w, r) {
			return
		}

		request := checkRequest(crudApp, rendererApp, authApp, w, r)
		if request != nil {
			handleRequest(w, r, crudApp, request)
		}
	})
}

func main() {
	fs := flag.NewFlagSet("fibr", flag.ExitOnError)

	serverConfig := httputils.Flags(fs, "")
	alcotestConfig := alcotest.Flags(fs, "")
	prometheusConfig := prometheus.Flags(fs, "prometheus")
	opentracingConfig := opentracing.Flags(fs, "tracing")
	owaspConfig := owasp.Flags(fs, "")

	authConfig := auth.Flags(fs, "auth")
	basicConfig := basic.Flags(fs, "basic")
	crudConfig := crud.Flags(fs, "")
	rendererConfig := renderer.Flags(fs, "")

	filesystemConfig := filesystem.Flags(fs, "fs")
	thumbnailConfig := thumbnail.Flags(fs, "thumbnail")

	if err := fs.Parse(os.Args[1:]); err != nil {
		logger.Fatal("%+v", err)
	}

	alcotest.DoAndExit(alcotestConfig)

	storage, err := filesystem.New(filesystemConfig)
	if err != nil {
		logger.Error("%+v", err)
		os.Exit(1)
	}

	serverApp := httputils.New(serverConfig)
	healthcheckApp := healthcheck.New()
	prometheusApp := prometheus.New(prometheusConfig)
	opentracingApp := opentracing.New(opentracingConfig)
	owaspApp := owasp.New(owaspConfig)
	gzipApp := gzip.New()

	thumbnailApp := thumbnail.New(thumbnailConfig, storage)
	rendererApp := renderer.New(rendererConfig, storage.Root(), thumbnailApp)
	crudApp := crud.New(crudConfig, storage, rendererApp, thumbnailApp)
	authApp := auth.NewService(authConfig, authService.NewBasic(basicConfig, nil))

	webHandler := server.ChainMiddlewares(browserHandler(crudApp, rendererApp, authApp), prometheusApp, opentracingApp, gzipApp, owaspApp)

	serverApp.ListenAndServe(webHandler, nil, healthcheckApp)
}
