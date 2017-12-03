package main

import (
	"bytes"
	"flag"
	"fmt"
	"html/template"
	"io/ioutil"
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
	"github.com/tdewolff/minify/css"
	"github.com/tdewolff/minify/html"
	"github.com/tdewolff/minify/js"
	"github.com/tdewolff/minify/svg"
)

var serviceHandler http.Handler
var apiHandler http.Handler
var tpl *template.Template
var minifier *minify.M

type config struct {
	StaticURL string
	AuthURL   string
	Version   string
}

type seo struct {
	Title       string
	Description string
	URL         string
	Img         string
	ImgHeight   uint
	ImgWidth    uint
}

type page struct {
	Config  *config
	Seo     *seo
	Current os.FileInfo
	Files   []os.FileInfo
	Login   bool
}

func init() {
	tpl = template.Must(template.New(`fibr`).Funcs(template.FuncMap{
		`filename`: func(file os.FileInfo) string {
			if file.IsDir() {
				return fmt.Sprintf(`%s/`, file.Name())
			}
			return file.Name()
		},
		`parent`: func(fullPath string, file os.FileInfo) string {
			return strings.TrimSuffix(fullPath, file.Name()+`/`)
		},
	}).ParseGlob(`./web/*.html`))
	tpl = template.Must(tpl.ParseGlob(`./web/svg/*.html`))

	minifier = minify.New()
	minifier.AddFunc("text/css", css.Minify)
	minifier.AddFunc("text/html", html.Minify)
	minifier.AddFunc("text/javascript", js.Minify)
	minifier.AddFunc("image/svg+xml", svg.Minify)
}

func getPathInfo(parts ...string) (string, os.FileInfo) {
	fullPath := path.Join(parts...)
	info, err := os.Stat(fullPath)

	if err != nil {
		return fullPath, nil
	}
	return fullPath, info
}

func writePageTemplate(w http.ResponseWriter, content *page) error {
	templateBuffer := &bytes.Buffer{}
	if err := tpl.ExecuteTemplate(templateBuffer, `page`, content); err != nil {
		return err
	}

	w.Header().Add(`Content-Type`, `text/html`)
	minifier.Minify(`text/html`, w, templateBuffer)
	return nil
}

func browserHandler(directory string, staticURL string, authConfig map[string]*string) http.Handler {
	templateConfig := config{
		StaticURL: staticURL,
		AuthURL:   *authConfig[`url`],
	}

	return auth.Handler(*authConfig[`url`], auth.LoadUsersProfiles(*authConfig[`users`]), func(w http.ResponseWriter, r *http.Request, user *auth.User) {
		filename, info := getPathInfo(directory, r.URL.Path)

		if info == nil {
			httputils.NotFound(w)
		} else if info.IsDir() {
			files, err := ioutil.ReadDir(filename)
			if err != nil {
				httputils.InternalServerError(w, err)
				return
			}

			content := page{
				Config: &templateConfig,
				Seo: &seo{
					Title:       fmt.Sprintf(`fibr - %s`, r.URL.Path),
					Description: fmt.Sprintf(`FIle BRowser of directory %s on the server`, r.URL.Path),
					URL:         r.URL.Path,
					Img:         staticURL + `/favicon/android-chrome-512x512.png`,
					ImgHeight:   512,
					ImgWidth:    512,
				},
				Current: info,
				Files:   files,
			}

			if err := writePageTemplate(w, &content); err != nil {
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
	directory := flag.String(`directory`, `/data/`, `Directory to serve`)
	staticURL := flag.String(`staticURL`, `https://fibr-static.vibioh.fr`, `Static Server URL`)
	authConfig := auth.Flags(`auth`)
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

	serviceHandler = owasp.Handler(owaspConfig, browserHandler(*directory, *staticURL, authConfig))
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
