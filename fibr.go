package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/NYTimes/gziphandler"
	"github.com/ViBiOh/alcotest/alcotest"
	"github.com/ViBiOh/auth/auth"
	"github.com/ViBiOh/fibr/crud"
	"github.com/ViBiOh/fibr/ui"
	"github.com/ViBiOh/fibr/utils"
	"github.com/ViBiOh/httputils"
	"github.com/ViBiOh/httputils/cert"
	"github.com/ViBiOh/httputils/owasp"
	"github.com/ViBiOh/httputils/prometheus"
	"github.com/ViBiOh/httputils/rate"
)

const metadataFileName = `.fibr_meta`

type share struct {
	id     string
	path   string
	public bool
	edit   bool
}

type metadata struct {
	shared map[string]share
}

var (
	serviceHandler http.Handler
	apiHandler     http.Handler
	meta           metadata
)

func handleAnonymousRequest(w http.ResponseWriter, r *http.Request, err error, uiConfig *ui.Config) {
	if auth.IsForbiddenErr(err) {
		uiConfig.Error(w, http.StatusForbidden, errors.New(`You're not authorized to do this`))
	} else if !crud.CheckAndServeSEO(w, r, uiConfig) {
		uiConfig.Login(w, nil)
	}
}

func browserHandler(directory string, uiConfig *ui.Config, authConfig map[string]*string) http.Handler {
	url := *authConfig[`url`]
	users := auth.LoadUsersProfiles(*authConfig[`users`])

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodPut && r.Method != http.MethodPost && r.Method != http.MethodDelete {
			uiConfig.Error(w, http.StatusMethodNotAllowed, errors.New(`We don't understand what you want from us`))
			return
		}

		_, err := auth.IsAuthenticated(url, users, r)

		if err != nil {
			handleAnonymousRequest(w, r, err, uiConfig)
		} else if r.Method == http.MethodGet {
			crud.Get(w, r, directory, uiConfig)
		} else if r.Method == http.MethodPut {
			crud.CreateDir(w, r, directory, uiConfig)
		} else if r.Method == http.MethodPost {
			crud.SaveFile(w, r, directory, uiConfig)
		} else if r.Method == http.MethodDelete {
			crud.Delete(w, r, directory, uiConfig)
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

func loadMetadata() error {
	rawMeta, err := ioutil.ReadFile(metadataFileName)
	if err != nil {
		return fmt.Errorf(`Error while reading metadata: %v`, err)
	}

	if err = json.Unmarshal(rawMeta, &meta); err != nil {
		return fmt.Errorf(`Error while unmarshalling metadata: %v`, err)
	}

	return nil
}

func saveMetadata() error {
	content, err := json.Marshal(&meta)
	if err != nil {
		return fmt.Errorf(`Error while marshalling metadata: %v`, err)
	}

	if err := ioutil.WriteFile(metadataFileName, content, 0600); err != nil {
		return fmt.Errorf(`Error while writing metadata: %v`, err)
	}

	return nil
}

func main() {
	port := flag.String(`port`, `1080`, `Listening port`)
	tls := flag.Bool(`tls`, true, `Serve TLS content`)
	directory := flag.String(`directory`, `/data`, `Directory to serve`)
	publicURL := flag.String(`publicURL`, `https://fibr.vibioh.fr`, `Public Server URL`)
	staticURL := flag.String(`staticURL`, `https://fibr-static.vibioh.fr`, `Static Server URL`)
	version := flag.String(`version`, ``, `Version (used mainly as a cache-buster)`)
	authConfig := auth.Flags(`auth`)
	alcotestConfig := alcotest.Flags(``)
	certConfig := cert.Flags(`tls`)
	prometheusConfig := prometheus.Flags(`prometheus`)
	rateConfig := rate.Flags(`rate`)
	owaspConfig := owasp.Flags(``)
	flag.Parse()

	alcotest.DoAndExit(alcotestConfig)

	_, info := utils.GetPathInfo(*directory)
	if info == nil || !info.IsDir() {
		log.Fatalf(`Directory %s is unreachable`, *directory)
	}

	if err := loadMetadata(); err != nil {
		log.Printf(`Error while loading metadata: %v`, err)
	}

	uiConfig := ui.NewUI(*publicURL, *staticURL, *authConfig[`url`], *version, *directory, info.Name())

	log.Printf(`Starting server on port %s`, *port)
	log.Printf(`Serving file from %s`, *directory)

	serviceHandler = owasp.Handler(owaspConfig, browserHandler(*directory, uiConfig, authConfig))
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
