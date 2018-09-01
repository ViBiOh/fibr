package main

import (
	"encoding/base64"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/ViBiOh/auth/pkg/auth"
	authProvider "github.com/ViBiOh/auth/pkg/provider"
	"github.com/ViBiOh/auth/pkg/provider/basic"
	authService "github.com/ViBiOh/auth/pkg/service"
	"github.com/ViBiOh/fibr/pkg/crud"
	"github.com/ViBiOh/fibr/pkg/filesystem"
	"github.com/ViBiOh/fibr/pkg/minio"
	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/fibr/pkg/thumbnail"
	"github.com/ViBiOh/fibr/pkg/ui"
	"github.com/ViBiOh/httputils/pkg"
	"github.com/ViBiOh/httputils/pkg/alcotest"
	"github.com/ViBiOh/httputils/pkg/gzip"
	"github.com/ViBiOh/httputils/pkg/healthcheck"
	"github.com/ViBiOh/httputils/pkg/httperror"
	"github.com/ViBiOh/httputils/pkg/opentracing"
	"github.com/ViBiOh/httputils/pkg/owasp"
	"github.com/ViBiOh/httputils/pkg/rollbar"
	"github.com/ViBiOh/httputils/pkg/server"
	"golang.org/x/crypto/bcrypt"
)

var errEmptyAuthorizationHeader = errors.New(`Empty authorization header`)

func checkSharePassword(r *http.Request, share *provider.Share) error {
	header := r.Header.Get(`Authorization`)
	if header == `` {
		return errEmptyAuthorizationHeader
	}

	data, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(header, `Basic `))
	if err != nil {
		return fmt.Errorf(`Error while decoding basic authentication: %v`, err)
	}

	dataStr := string(data)

	sepIndex := strings.Index(dataStr, `:`)
	if sepIndex < 0 {
		return errors.New(`Error while reading basic authentication`)
	}

	password := dataStr[sepIndex+1:]
	if err := bcrypt.CompareHashAndPassword([]byte(share.Password), []byte(password)); err != nil {
		return errors.New(`Invalid credentials`)
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
	if auth.IsForbiddenErr(err) {
		uiApp.Error(w, http.StatusForbidden, errors.New(`You're not authorized to do this ⛔️`))
		return
	}

	if err == authProvider.ErrMalformedAuth || err == authProvider.ErrUnknownAuthType {
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
			uiApp.Error(w, http.StatusMethodNotAllowed, errors.New(`We don't understand what you want from us`))
			return
		}

		if strings.Contains(r.URL.Path, `..`) {
			uiApp.Error(w, http.StatusForbidden, errors.New(`You're not authorized to do this ⛔`))
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

func getStorage(storageName string, configs map[string]interface{}) provider.Storage {
	config, ok := configs[storageName]
	if !ok {
		log.Fatalf(`Unable to find storage config %s`, storageName)
		return nil
	}

	var app provider.Storage
	var appNilValue provider.Storage
	var err error

	switch storageName {
	case `filesystem`:
		appNilValue = (*filesystem.App)(nil)
		app, err = filesystem.NewApp(config.(map[string]*string))

	default:
		err = errors.New(`Unknown storage type`)
	}

	if err != nil {
		log.Fatalf(`Error while initializing storage: %v`, err)
		return nil
	}

	if app == appNilValue {
		log.Fatalf(`Unable to initialize storage: %v`, err)
		return nil
	}

	return app
}

func main() {
	serverConfig := httputils.Flags(``)
	alcotestConfig := alcotest.Flags(``)
	opentracingConfig := opentracing.Flags(`tracing`)
	owaspConfig := owasp.Flags(``)
	rollbarConfig := rollbar.Flags(`rollbar`)

	authConfig := auth.Flags(`auth`)
	basicConfig := basic.Flags(`basic`)
	crudConfig := crud.Flags(``)
	uiConfig := ui.Flags(``)

	filesystemConfig := filesystem.Flags(`fs`)
	minio.Flags(`minio`)

	flag.Parse()

	alcotest.DoAndExit(alcotestConfig)

	serverApp := httputils.NewApp(serverConfig)
	healthcheckApp := healthcheck.NewApp()
	opentracingApp := opentracing.NewApp(opentracingConfig)
	owaspApp := owasp.NewApp(owaspConfig)
	rollbarApp := rollbar.NewApp(rollbarConfig)
	gzipApp := gzip.NewApp()

	storage := getStorage(`filesystem`, map[string]interface{}{
		`filesystem`: filesystemConfig,
	})

	thumbnailApp := thumbnail.NewApp(storage)
	uiApp := ui.NewApp(uiConfig, storage.Root(), thumbnailApp)
	crudApp := crud.NewApp(crudConfig, storage, uiApp, thumbnailApp)
	authApp := auth.NewApp(authConfig, authService.NewBasicApp(basicConfig))

	webHandler := server.ChainMiddlewares(browserHandler(crudApp, uiApp, authApp), opentracingApp, rollbarApp, gzipApp, owaspApp)

	serverApp.ListenAndServe(webHandler, nil, healthcheckApp)
}
