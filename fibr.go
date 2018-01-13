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

func handleAnonymousRequest(w http.ResponseWriter, r *http.Request, err error, uiApp *ui.App) {
	if auth.IsForbiddenErr(err) {
		uiApp.Error(w, http.StatusForbidden, errors.New(`You're not authorized to do this`))
	} else if !crud.CheckAndServeSEO(w, r, uiApp) {
		uiApp.Login(w, nil)
	}
}

func browserHandler(directory string, uiApp *ui.App, authConfig map[string]*string) http.Handler {
	url := *authConfig[`url`]
	users := auth.LoadUsersProfiles(*authConfig[`users`])

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodPut && r.Method != http.MethodPost && r.Method != http.MethodDelete {
			uiApp.Error(w, http.StatusMethodNotAllowed, errors.New(`We don't understand what you want from us`))
			return
		}

		_, err := auth.IsAuthenticated(url, users, r)

		if err != nil {
			handleAnonymousRequest(w, r, err, uiApp)
		} else if r.Method == http.MethodGet {
			crud.Get(w, r, directory, uiApp)
		} else if r.Method == http.MethodPut {
			crud.CreateDir(w, r, directory, uiApp)
		} else if r.Method == http.MethodPost {
			crud.SaveFile(w, r, directory, uiApp)
		} else if r.Method == http.MethodDelete {
			crud.Delete(w, r, directory, uiApp)
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
	publicURL := flag.String(`publicURL`, `https://fibr.vibioh.fr`, `Public Server URL`)
	staticURL := flag.String(`staticURL`, `https://fibr-static.vibioh.fr`, `Static Server URL`)
	version := flag.String(`version`, ``, `Version (used mainly as a cache-buster)`)
	authConfig := auth.Flags(`auth`)
	alcotestConfig := alcotest.Flags(``)
	certConfig := cert.Flags(`tls`)
	prometheusConfig := prometheus.Flags(`prometheus`)
	rateConfig := rate.Flags(`rate`)
	owaspConfig := owasp.Flags(``)
	crudConfig := crud.Flags(``)
	flag.Parse()

	alcotest.DoAndExit(alcotestConfig)

	crudApp := crud.NewApp(crudConfig)
	uiApp := ui.NewApp(*publicURL, *staticURL, *authConfig[`url`], *version, crudApp.GetRootAbsName(), crudApp.GetRootName())

	log.Printf(`Starting server on port %s`, *port)
	log.Printf(`Serving file from %s`, crudApp.GetRootAbsName())

	serviceHandler = owasp.Handler(owaspConfig, browserHandler(crudApp.GetRootAbsName(), uiApp, authConfig))
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
