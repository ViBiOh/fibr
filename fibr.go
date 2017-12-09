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
	Config    *config
	Seo       *seo
	Current   os.FileInfo
	PathParts []string
	Files     []os.FileInfo
	Login     bool
}

var templateConfig *config
var seoConfig *seo

func init() {
	tpl = template.Must(template.New(`fibr`).Funcs(template.FuncMap{
		`filename`: func(file os.FileInfo) string {
			if file.IsDir() {
				return fmt.Sprintf(`%s/`, file.Name())
			}
			return file.Name()
		},
		`rebuildPaths`: func(parts []string, index int) string {
			return path.Join(parts[:index+1]...)
		},
	}).ParseGlob(`./web/*.html`))

	minifier = minify.New()
	minifier.AddFunc("text/css", css.Minify)
	minifier.AddFunc("text/html", html.Minify)
	minifier.AddFunc("text/javascript", js.Minify)
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

func createPage(path string, current os.FileInfo, files []os.FileInfo, login bool) *page {
	pathParts := strings.Split(strings.Trim(path, `/`), `/`)
	if pathParts[0] == `` {
		pathParts = nil
	}

	return &page{
		Config: templateConfig,
		Seo: &seo{
			Title:       fmt.Sprintf(`fibr - %s`, path),
			Description: fmt.Sprintf(`FIle BRowser of directory %s on the server`, path),
			URL:         path,
			Img:         seoConfig.Img,
			ImgHeight:   seoConfig.ImgHeight,
			ImgWidth:    seoConfig.ImgWidth,
		},
		PathParts: pathParts,
		Current:   current,
		Files:     files,
		Login:     login,
	}
}

func browserHandler(directory string, authConfig map[string]*string) http.Handler {
	return auth.HandlerWithFail(*authConfig[`url`], auth.LoadUsersProfiles(*authConfig[`users`]), func(w http.ResponseWriter, r *http.Request, user *auth.User) {
		filename, info := getPathInfo(directory, r.URL.Path)

		if info == nil {
			httputils.NotFound(w)
		} else if info.IsDir() {
			files, err := ioutil.ReadDir(filename)
			if err != nil {
				httputils.InternalServerError(w, err)
				return
			}

			if err := writePageTemplate(w, createPage(r.URL.Path, info, files, false)); err != nil {
				httputils.InternalServerError(w, err)
			}
		} else {
			http.ServeFile(w, r, filename)
		}
	}, func(w http.ResponseWriter, r *http.Request, err error) {
		if auth.IsForbiddenErr(err) {
			httputils.Forbidden(w)
		} else {
			if err := writePageTemplate(w, createPage(r.URL.Path, nil, nil, true)); err != nil {
				httputils.InternalServerError(w, err)
			}
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

func initTemplateConfiguration(staticURL string, authURL string, version string) {
	templateConfig = &config{
		StaticURL: staticURL,
		AuthURL:   authURL,
		Version:   version,
	}

	seoConfig = &seo{
		Title:     `fibr`,
		URL:       `/`,
		Img:       staticURL + `/favicon/android-chrome-512x512.png`,
		ImgHeight: 512,
		ImgWidth:  512,
	}
}

func main() {
	port := flag.String(`port`, `1080`, `Listening port`)
	tls := flag.Bool(`tls`, true, `Serve TLS content`)
	directory := flag.String(`directory`, `/data/`, `Directory to serve`)
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

	if _, info := getPathInfo(*directory); info == nil || !info.IsDir() {
		log.Fatalf(`Directory %s is unreachable`, *directory)
	}

	log.Printf(`Starting server on port %s`, *port)
	log.Printf(`Serving file from %s`, *directory)

	initTemplateConfiguration(*staticURL, *authConfig[`url`], *version)

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
