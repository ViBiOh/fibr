package main

import (
	"bytes"
	"flag"
	"html/template"
	"log"
	"net/http"
	"os"
	"path"

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

func getPathInfo(parts ...string) (string, os.FileInfo) {
	fullPath := path.Join(parts...)
	info, err := os.Stat(fullPath)

	if err != nil {
		return fullPath, nil
	}
	return fullPath, info
}

func writePageTemplate(w http.ResponseWriter) error {
	templateBuffer := &bytes.Buffer{}
	if err := tpl.ExecuteTemplate(templateBuffer, `page`, nil); err != nil {
		return err
	}

	w.Header().Add(`Content-Type`, `text/html`)
	minifier.Minify(`text/html`, w, templateBuffer)
	return nil
}

func browserHandler(directory string, authConfig map[string]*string) http.Handler {
	return auth.Handler(*authConfig[`url`], auth.LoadUsersProfiles(*authConfig[`users`]), func(w http.ResponseWriter, r *http.Request, user *auth.User) {
		filename, info := getPathInfo(directory, r.URL.Path)

		if info == nil {
			httputils.NotFound(w)
		} else if info.IsDir() {
			if err := writePageTemplate(w); err != nil {
				httputils.InternalServerError(w, err)
			}
		} else {
			http.ServeFile(w, r, filename)
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
	directory := flag.String(`directory`, `/data/`, `Directory to serve`)
	alcotestConfig := alcotest.Flags(``)
	certConfig := cert.Flags(`tls`)
	prometheusConfig := prometheus.Flags(`prometheus`)
	rateConfig := rate.Flags(`rate`)
	owaspConfig := owasp.Flags(``)
	flag.Parse()

	alcotest.DoAndExit(alcotestConfig)

	if _, info := getPathInfo(*directory); info == nil || !info.IsDir() {
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
