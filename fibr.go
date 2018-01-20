package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"path"
	"strings"

	"github.com/NYTimes/gziphandler"
	"github.com/ViBiOh/alcotest/alcotest"
	"github.com/ViBiOh/auth/auth"
	authProvider "github.com/ViBiOh/auth/provider"
	"github.com/ViBiOh/auth/provider/basic"
	"github.com/ViBiOh/auth/service"
	"github.com/ViBiOh/fibr/crud"
	"github.com/ViBiOh/fibr/provider"
	"github.com/ViBiOh/fibr/ui"
	"github.com/ViBiOh/httputils"
	"github.com/ViBiOh/httputils/cert"
	"github.com/ViBiOh/httputils/owasp"
	"github.com/ViBiOh/httputils/prometheus"
	"github.com/ViBiOh/httputils/rate"
)

var (
	serviceHandler http.Handler
	apiHandler     http.Handler
)

func handleAnonymousRequest(w http.ResponseWriter, r *http.Request, err error, crudApp *crud.App, uiApp *ui.App) {
	if auth.IsForbiddenErr(err) {
		uiApp.Error(w, http.StatusForbidden, errors.New(`You're not authorized to do this`))
	} else if !crudApp.CheckAndServeSEO(w, r) {
		if err == authProvider.ErrMalformedAuth || err == authProvider.ErrUnknownAuthType {
			uiApp.Error(w, http.StatusBadRequest, err)
		} else {
			w.Header().Add(`WWW-Authenticate`, `Basic`)
			uiApp.Error(w, http.StatusUnauthorized, err)
		}
	}
}

func browserHandler(crudApp *crud.App, uiApp *ui.App, authApp *auth.App) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodPost && r.Method != http.MethodPut && r.Method != http.MethodDelete {
			uiApp.Error(w, http.StatusMethodNotAllowed, errors.New(`We don't understand what you want from us`))
			return
		}

		_, err := authApp.IsAuthenticated(r)

		rootDirectory := crudApp.GetRootDirectory()
		config := &provider.RequestConfig{
			URL:        r.URL.Path,
			Root:       rootDirectory,
			PathPrefix: ``,
			Path:       r.URL.Path,
			CanEdit:    true,
			CanShare:   true,
		}

		if share := crudApp.GetSharedPath(config.Path); share != nil {
			config.Root = path.Join(config.Root, share.Path)
			config.Path = strings.TrimPrefix(config.Path, fmt.Sprintf(`/%s`, share.ID))
			config.PathPrefix = share.ID

			if err != nil {
				config.CanEdit = share.Edit
				config.CanShare = false
			}

			err = nil
		}

		if err != nil {
			handleAnonymousRequest(w, r, err, crudApp, uiApp)
		} else if r.Method == http.MethodGet {
			crudApp.Get(w, r, config, nil)
		} else if r.Method == http.MethodPost {
			crudApp.Post(w, r, config)
		} else if r.Method == http.MethodPut {
			crudApp.CreateDir(w, r, config)
		} else if r.Method == http.MethodDelete {
			crudApp.Delete(w, r, config)
		} else {
			httputils.NotFound(w)
		}
	})
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == `/health` {
			healthHandler(w, r)
		} else {
			serviceHandler.ServeHTTP(w, r)
		}
	})
}

func main() {
	port := flag.String(`port`, `1080`, `Listening port`)
	tls := flag.Bool(`tls`, true, `Serve TLS content`)
	alcotestConfig := alcotest.Flags(``)
	certConfig := cert.Flags(`tls`)
	prometheusConfig := prometheus.Flags(`prometheus`)
	rateConfig := rate.Flags(`rate`)
	owaspConfig := owasp.Flags(``)

	authConfig := auth.Flags(`auth`)
	serviceAuthConfig := service.Flags(`authService`)
	basicConfig := basic.Flags(`basic`)

	crudConfig := crud.Flags(``)
	uiConfig := ui.Flags(``)

	flag.Parse()

	alcotest.DoAndExit(alcotestConfig)

	log.Printf(`Starting server on port %s`, *port)

	authApp := auth.NewApp(authConfig, service.NewApp(serviceAuthConfig, basicConfig, nil))
	uiApp := ui.NewApp(uiConfig, authApp.URL)
	crudApp := crud.NewApp(crudConfig, uiApp)

	serviceHandler = owasp.Handler(owaspConfig, browserHandler(crudApp, uiApp, authApp))
	apiHandler = prometheus.Handler(prometheusConfig, rate.Handler(rateConfig, gziphandler.GzipHandler(handler())))

	server := &http.Server{
		Addr:    `:` + *port,
		Handler: apiHandler,
	}

	var serveError = make(chan error)
	go func() {
		defer close(serveError)
		if *tls {
			log.Print(`Listening with TLS enabled`)
			serveError <- cert.ListenAndServeTLS(certConfig, server)
		} else {
			log.Print(`⚠ fibr is running without secure connection ⚠`)
			serveError <- server.ListenAndServe()
		}
	}()

	httputils.ServerGracefulClose(server, serveError, nil)
}
