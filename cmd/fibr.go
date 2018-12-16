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
	"github.com/ViBiOh/fibr/pkg/minio"
	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/fibr/pkg/thumbnail"
	"github.com/ViBiOh/fibr/pkg/ui"
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

var errEmptyAuthorizationHeader = errors.New(`empty authorization header`)

func checkSharePassword(r *http.Request, share *provider.Share) error {
	header := r.Header.Get(`Authorization`)
	if header == `` {
		return errEmptyAuthorizationHeader
	}

	data, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(header, `Basic `))
	if err != nil {
		return errors.WithStack(err)
	}

	dataStr := string(data)

	sepIndex := strings.Index(dataStr, `:`)
	if sepIndex < 0 {
		return errors.New(`invalid format for basic auth`)
	}

	password := dataStr[sepIndex+1:]
	if err := bcrypt.CompareHashAndPassword([]byte(share.Password), []byte(password)); err != nil {
		return errors.New(`invalid credentials`)
	}

	return nil
}

func checkShare(w http.ResponseWriter, r *http.Request, crudApp *crud.App, request *provider.Request) error {
	if share := crudApp.GetShare(request.Path); share != nil {
		request.Share = share
		request.CanEdit = share.Edit
		request.Path = strings.TrimPrefix(request.Path, fmt.Sprintf(`/%s`, share.ID))

		if share.Password != `` {
			if err := checkSharePassword(r, share); err != nil {
				w.Header().Add(`WWW-Authenticate`, `Basic realm="Password required" charset="UTF-8"`)
				return err
			}
		}
	}

	return nil
}

func handleAnonymousRequest(w http.ResponseWriter, r *http.Request, err error, crudApp *crud.App, uiApp *ui.App) {
	if auth.ErrForbidden == err {
		uiApp.Error(w, http.StatusForbidden, errors.New(`you're not authorized to do this ⛔️`))
		return
	}

	if err == ident.ErrMalformedAuth || err == ident.ErrUnknownIdentType {
		uiApp.Error(w, http.StatusBadRequest, err)
		return
	}

	w.Header().Add(`WWW-Authenticate`, `Basic charset="UTF-8"`)
	uiApp.Error(w, http.StatusUnauthorized, err)
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

func checkAllowedMethod(r *http.Request) bool {
	return r.Method == http.MethodGet || r.Method == http.MethodPost || r.Method == http.MethodPut || r.Method == http.MethodPatch || r.Method == http.MethodDelete
}

func browserHandler(crudApp *crud.App, uiApp *ui.App, authApp *auth.App) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !checkAllowedMethod(r) {
			uiApp.Error(w, http.StatusMethodNotAllowed, errors.New(`we don't understand what you want from us`))
			return
		}

		if strings.Contains(r.URL.Path, `..`) {
			uiApp.Error(w, http.StatusForbidden, errors.New(`you're not authorized to do this ⛔`))
			return
		}

		if crudApp.CheckAndServeSEO(w, r) {
			return
		}

		request := &provider.Request{
			Path:     r.URL.Path,
			CanEdit:  false,
			CanShare: false,
			IsDebug:  r.URL.Query().Get(`debug`) == `true`,
		}

		if err := checkShare(w, r, crudApp, request); err != nil {
			uiApp.Error(w, http.StatusUnauthorized, err)
			return
		}

		if request.Share == nil {
			user, err := authApp.IsAuthenticated(r)
			if err != nil {
				handleAnonymousRequest(w, r, err, crudApp, uiApp)
				return
			}

			if user != nil && user.HasProfile(`admin`) {
				request.CanEdit = true
				request.CanShare = true
			}
		}

		handleRequest(w, r, crudApp, request)
	})
}

func getStorage(storageName string, configs map[string]map[string]*string) (provider.Storage, error) {
	config, ok := configs[storageName]
	if !ok {
		return nil, errors.New(`no storage config for %s`, storageName)
	}

	var app provider.Storage
	var err error

	switch storageName {
	case `filesystem`:
		app, err = filesystem.NewApp(config)

	default:
		err = errors.New(`unknown storage type`)
	}

	if err != nil {
		return nil, err
	}

	return app, nil
}

func main() {
	serverConfig := httputils.Flags(``)
	alcotestConfig := alcotest.Flags(``)
	prometheusConfig := prometheus.Flags(`prometheus`)
	opentracingConfig := opentracing.Flags(`tracing`)
	owaspConfig := owasp.Flags(``)

	authConfig := auth.Flags(`auth`)
	basicConfig := basic.Flags(`basic`)
	crudConfig := crud.Flags(``)
	uiConfig := ui.Flags(``)

	filesystemConfig := filesystem.Flags(`fs`)
	minioConfig := minio.Flags(`minio`)

	storageName := flag.String(`storage`, `filesystem`, `Storage used (e.g. 'filesystem', 'minio')`)

	flag.Parse()

	alcotest.DoAndExit(alcotestConfig)

	storage, err := getStorage(*storageName, map[string]map[string]*string{
		`filesystem`: filesystemConfig,
		`minio`:      minioConfig,
	})
	if err != nil {
		logger.Error(`%+v`, err)
		os.Exit(1)
	}

	serverApp := httputils.NewApp(serverConfig)
	healthcheckApp := healthcheck.NewApp()
	prometheusApp := prometheus.NewApp(prometheusConfig)
	opentracingApp := opentracing.NewApp(opentracingConfig)
	owaspApp := owasp.NewApp(owaspConfig)
	gzipApp := gzip.NewApp()

	thumbnailApp := thumbnail.NewApp(storage)
	uiApp := ui.NewApp(uiConfig, storage.Root(), thumbnailApp)
	crudApp := crud.NewApp(crudConfig, storage, uiApp, thumbnailApp)
	authApp := auth.NewServiceApp(authConfig, authService.NewBasicApp(basicConfig, nil))

	webHandler := server.ChainMiddlewares(browserHandler(crudApp, uiApp, authApp), prometheusApp, opentracingApp, gzipApp, owaspApp)

	serverApp.ListenAndServe(webHandler, nil, healthcheckApp)
}
