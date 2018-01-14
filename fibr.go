package main

import (
	"errors"
	"flag"
	"log"
	"net/http"

	"github.com/NYTimes/gziphandler"
	"github.com/ViBiOh/alcotest/alcotest"
	"github.com/ViBiOh/auth/auth"
	"github.com/ViBiOh/fibr/crud"
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
		uiApp.Login(w, nil)
	}
}

func browserHandler(crudApp *crud.App, uiApp *ui.App, authConfig map[string]*string) http.Handler {
	url := *authConfig[`url`]
	users := auth.LoadUsersProfiles(*authConfig[`users`])

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodPut && r.Method != http.MethodPost && r.Method != http.MethodDelete {
			uiApp.Error(w, http.StatusMethodNotAllowed, errors.New(`We don't understand what you want from us`))
			return
		}

		_, err := auth.IsAuthenticated(url, users, r)

		rootDirectory := crudApp.GetRootDirectory()

		if err != nil {
			handleAnonymousRequest(w, r, err, crudApp, uiApp)
		} else if r.Method == http.MethodGet {
			crudApp.Get(w, r, rootDirectory)
		} else if r.Method == http.MethodPut {
			crudApp.CreateDir(w, r, rootDirectory)
		} else if r.Method == http.MethodPost {
			crudApp.SaveFile(w, r, rootDirectory)
		} else if r.Method == http.MethodDelete {
			crudApp.Delete(w, r, rootDirectory)
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
	authConfig := auth.Flags(`auth`)
	alcotestConfig := alcotest.Flags(``)
	certConfig := cert.Flags(`tls`)
	prometheusConfig := prometheus.Flags(`prometheus`)
	rateConfig := rate.Flags(`rate`)
	owaspConfig := owasp.Flags(``)

	crudConfig := crud.Flags(``)
	uiConfig := ui.Flags(``)

	flag.Parse()

	alcotest.DoAndExit(alcotestConfig)

	uiApp := ui.NewApp(uiConfig, *authConfig[`url`])
	crudApp := crud.NewApp(crudConfig, uiApp)

	log.Printf(`Starting server on port %s`, *port)

	serviceHandler = owasp.Handler(owaspConfig, browserHandler(crudApp, uiApp, authConfig))
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
