package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"log"
	"mime/multipart"
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

type share struct {
	id     string
	path   string
	public bool
}

type metadata struct {
	shared map[string]share
}

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

const metadataFileName = `.fibr_meta`
const maxUploadSize = 32 * 1024 * 2014 // 32 MB

var archiveExtension = map[string]bool{`.zip`: true, `.tar`: true, `.gz`: true}
var audioExtension = map[string]bool{`.mp3`: true}
var codeExtension = map[string]bool{`.html`: true, `.css`: true, `.js`: true, `.jsx`: true, `.json`: true, `.yml`: true, `.yaml`: true, `.toml`: true, `.md`: true, `.go`: true}
var excelExtension = map[string]bool{`.xls`: true, `.xlsx`: true, `.xlsm`: true}
var imageExtension = map[string]bool{`.jpg`: true, `.jpeg`: true, `.png`: true, `.gif`: true, `.svg`: true, `.tiff`: true}
var pdfExtension = map[string]bool{`.pdf`: true}
var videoExtension = map[string]bool{`.mp4`: true, `.mov`: true, `.avi`: true}
var wordExtension = map[string]bool{`.doc`: true, `.docx`: true}

var serviceHandler http.Handler
var apiHandler http.Handler
var tpl *template.Template
var minifier *minify.M

var templateConfig *config
var seoConfig *seo
var meta metadata

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
		`typeFromExtension`: func(file os.FileInfo) string {
			extension := path.Ext(file.Name())

			switch {
			case archiveExtension[extension]:
				return `-archive`
			case audioExtension[extension]:
				return `-audio`
			case codeExtension[extension]:
				return `-code`
			case excelExtension[extension]:
				return `-excel`
			case imageExtension[extension]:
				return `-image`
			case pdfExtension[extension]:
				return `-pdf`
			case videoExtension[extension]:
				return `-video`
			case wordExtension[extension]:
				return `-word`
			default:
				return ``
			}
		},
	}).ParseGlob(`./web/*.gohtml`))

	minifier = minify.New()
	minifier.AddFunc(`text/css`, css.Minify)
	minifier.AddFunc(`text/html`, html.Minify)
	minifier.AddFunc(`text/javascript`, js.Minify)
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

	w.Header().Add(`Content-Type`, `text/html; charset=UTF-8`)
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

func checkAndServeStatic(w http.ResponseWriter, r *http.Request) bool {
	if r.URL.Path == `/robots.txt` || r.URL.Path == `/sitemap.xml` {
		http.ServeFile(w, r, path.Join(`web/static`, r.URL.Path))
		return true
	}

	return false
}

func handleAnonymousRequest(w http.ResponseWriter, r *http.Request, err error) {
	if auth.IsForbiddenErr(err) {
		httputils.Forbidden(w)
	} else if !checkAndServeStatic(w, r) {
		if err := writePageTemplate(w, createPage(r.URL.Path, nil, nil, true)); err != nil {
			httputils.InternalServerError(w, err)
		}
	}
}

func handleLoggedRequest(w http.ResponseWriter, r *http.Request, directory string) {
	filename, info := getPathInfo(directory, r.URL.Path)

	if info == nil {
		if !checkAndServeStatic(w, r) {
			httputils.NotFound(w)
		}
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
}

func handleUploadRequest(w http.ResponseWriter, r *http.Request, directory string) {
	var uploadedFile multipart.File
	var hostFile *os.File
	var err error

	r.Body = http.MaxBytesReader(w, r.Body, maxUploadSize)

	uploadedFile, _, err = r.FormFile(`file`)
	if uploadedFile != nil {
		defer uploadedFile.Close()
	}
	if err != nil {
		http.Error(w, fmt.Errorf(`Error while extracting file: %v`, err).Error(), http.StatusBadRequest)
		return
	}

	filename, info := getPathInfo(directory, r.URL.Path)

	if info == nil {
		hostFile, err = os.Create(filename)
	} else {
		hostFile, err = os.Open(filename)
	}

	if hostFile != nil {
		defer hostFile.Close()
	}
	if err != nil {
		http.Error(w, fmt.Errorf(`Error while creating/opening file: %v`, err).Error(), http.StatusInternalServerError)
		return
	}

	_, err = io.Copy(hostFile, uploadedFile)
	if err != nil {
		http.Error(w, fmt.Errorf(`Error while writing file: %v`, err).Error(), http.StatusInternalServerError)
		return
	}
}

func browserHandler(directory string, authConfig map[string]*string) http.Handler {
	url := *authConfig[`url`]
	profiles := auth.LoadUsersProfiles(*authConfig[`users`])

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		_, err := auth.IsAuthenticated(url, profiles, r)

		if err != nil {
			handleAnonymousRequest(w, r, err)
		} else if r.Method == http.MethodGet {
			handleLoggedRequest(w, r, directory)
		} else if r.Method == http.MethodPost {
			handleUploadRequest(w, r, directory)
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

func initTemplateConfiguration(staticURL string, authURL string, version string) {
	templateConfig = &config{
		StaticURL: staticURL,
		AuthURL:   authURL,
		Version:   version,
	}

	seoConfig = &seo{
		Title:     `fibr`,
		URL:       `/`,
		Img:       path.Join(staticURL, `/favicon/android-chrome-512x512.png`),
		ImgHeight: 512,
		ImgWidth:  512,
	}
}

func loadMetadata() {
	rawMeta, err := ioutil.ReadFile(metadataFileName)
	if err != nil {
		log.Printf(`Error while reading metadata: %v`, err)
		return
	}

	if err = json.Unmarshal(rawMeta, &meta); err != nil {
		log.Printf(`Error while unmarshalling metadata: %v`, err)
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

	loadMetadata()

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
