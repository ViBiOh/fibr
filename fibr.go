package main

import (
	"bytes"
	"flag"
	"html/template"
	"log"
	"net/http"
	"os"
	"path"
	"strings"

	"github.com/NYTimes/gziphandler"
	"github.com/ViBiOh/alcotest/alcotest"
	"github.com/ViBiOh/auth/auth"
	"github.com/ViBiOh/httputils"
	"github.com/ViBiOh/httputils/cert"
	"github.com/ViBiOh/httputils/owasp"
	"github.com/ViBiOh/httputils/prometheus"
	"github.com/ViBiOh/httputils/rate"
	"github.com/tdewolff/minify"
	"github.com/tdewolff/minify/html"
)

var serviceHandler http.Handler
var apiHandler http.Handler
var tpl *template.Template
var minifier *minify.M

func init() {
	tpl = template.Must(template.ParseGlob(`./web/*.html`))
	minifier = minify.New()
	minifier.AddFunc("text/html", html.Minify)
}

func isFileExist(parts ...string) *string {
	fullPath := path.Join(parts...)

	if _, err := os.Stat(fullPath); err != nil {
		return nil
	}

	return &fullPath
}

func webHandler(w http.ResponseWriter, r *http.Request, user *auth.User, directory string) {
	templateBuffer := &bytes.Buffer{}
	if err := tpl.ExecuteTemplate(templateBuffer, `page`, nil); err != nil {
		httputils.InternalServerError(w, err)
	}

	w.Header().Add(`Content-Type`, `text/html`)
	minifier.Minify(`text/html`, w, templateBuffer)
}

func filesHandler(w http.ResponseWriter, r *http.Request, user *auth.User, directory string, path string) {
	if filename := isFileExist(directory, path); filename != nil {
		http.ServeFile(w, r, *filename)
	} else {
		httputils.NotFound(w)
	}
}

func browserHandler(directory string, authConfig map[string]*string) http.Handler {
	return auth.Handler(*authConfig[`url`], auth.LoadUsersProfiles(*authConfig[`users`]), func(w http.ResponseWriter, r *http.Request, user *auth.User) {
		if strings.HasPrefix(r.URL.Path, `/files`) {
			filesHandler(w, r, user, directory, strings.TrimPrefix(r.URL.Path, `/files`))
			return
		}
		webHandler(w, r, user, directory)
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
	directory := flag.String(`directory`, `/data/`, `Directory to serve`)
	alcotestConfig := alcotest.Flags(``)
	certConfig := cert.Flags(`tls`)
	prometheusConfig := prometheus.Flags(`prometheus`)
	rateConfig := rate.Flags(`rate`)
	owaspConfig := owasp.Flags(``)
	flag.Parse()

	alcotest.DoAndExit(alcotestConfig)

	if isFileExist(*directory) == nil {
		log.Fatalf(`Directory %s is unreachable`, *directory)
	}

	log.Printf(`Starting server on port %s`, *port)
	log.Printf(`Serving file from %s`, *directory)

	serviceHandler = owasp.Handler(owaspConfig, browserHandler(*directory, authConfig))
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
