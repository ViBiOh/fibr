package main

import (
	"embed"
	"flag"
	"os"

	"github.com/ViBiOh/auth/v2/pkg/ident/basic"
	authMiddleware "github.com/ViBiOh/auth/v2/pkg/middleware"
	basicMemory "github.com/ViBiOh/auth/v2/pkg/store/memory"
	"github.com/ViBiOh/fibr/pkg/crud"
	"github.com/ViBiOh/fibr/pkg/exif"
	"github.com/ViBiOh/fibr/pkg/fibr"
	"github.com/ViBiOh/fibr/pkg/filesystem"
	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/fibr/pkg/share"
	"github.com/ViBiOh/fibr/pkg/thumbnail"
	"github.com/ViBiOh/httputils/v4/pkg/alcotest"
	"github.com/ViBiOh/httputils/v4/pkg/flags"
	"github.com/ViBiOh/httputils/v4/pkg/health"
	"github.com/ViBiOh/httputils/v4/pkg/httputils"
	"github.com/ViBiOh/httputils/v4/pkg/logger"
	"github.com/ViBiOh/httputils/v4/pkg/owasp"
	"github.com/ViBiOh/httputils/v4/pkg/prometheus"
	"github.com/ViBiOh/httputils/v4/pkg/recoverer"
	"github.com/ViBiOh/httputils/v4/pkg/renderer"
	"github.com/ViBiOh/httputils/v4/pkg/server"
)

//go:embed templates static
var content embed.FS

func newLoginApp(basicConfig basicMemory.Config) authMiddleware.App {
	basicApp, err := basicMemory.New(basicConfig)
	logger.Fatal(err)

	basicProviderProvider := basic.New(basicApp, "fibr")
	return authMiddleware.New(basicApp, basicProviderProvider)
}

func main() {
	fs := flag.NewFlagSet("fibr", flag.ExitOnError)

	appServerConfig := server.Flags(fs, "", flags.NewOverride("ReadTimeout", "2m"), flags.NewOverride("WriteTimeout", "2m"))
	promServerConfig := server.Flags(fs, "prometheus", flags.NewOverride("Port", 9090), flags.NewOverride("IdleTimeout", "10s"), flags.NewOverride("ShutdownTimeout", "5s"))
	healthConfig := health.Flags(fs, "")

	alcotestConfig := alcotest.Flags(fs, "")
	loggerConfig := logger.Flags(fs, "logger")
	prometheusConfig := prometheus.Flags(fs, "prometheus", flags.NewOverride("Gzip", false))
	owaspConfig := owasp.Flags(fs, "", flags.NewOverride("FrameOptions", "SAMEORIGIN"), flags.NewOverride("Csp", "default-src 'self'; base-uri 'self'; script-src 'unsafe-inline'; style-src 'unsafe-inline'; img-src 'self' data:"))

	basicConfig := basicMemory.Flags(fs, "auth")

	crudConfig := crud.Flags(fs, "")
	shareConfig := share.Flags(fs, "")
	rendererConfig := renderer.Flags(fs, "", flags.NewOverride("PublicURL", "https://fibr.vibioh.fr"), flags.NewOverride("Title", "fibr"))

	filesystemConfig := filesystem.Flags(fs, "fs")
	thumbnailConfig := thumbnail.Flags(fs, "thumbnail")
	exifConfig := exif.Flags(fs, "exif")

	disableAuth := flags.New("", "auth").Name("NoAuth").Default(false).Label("Disable basic authentification").ToBool(fs)

	logger.Fatal(fs.Parse(os.Args[1:]))

	alcotest.DoAndExit(alcotestConfig)
	logger.Global(logger.New(loggerConfig))
	defer logger.Close()

	appServer := server.New(appServerConfig)
	promServer := server.New(promServerConfig)
	prometheusApp := prometheus.New(prometheusConfig)
	healthApp := health.New(healthConfig)

	storageApp, err := filesystem.New(filesystemConfig)
	logger.Fatal(err)

	prometheusRegister := prometheusApp.Registerer()
	eventBus := provider.NewEventBus(10)

	thumbnailApp := thumbnail.New(thumbnailConfig, storageApp, prometheusRegister)
	exifApp := exif.New(exifConfig, storageApp, prometheusRegister)

	rendererApp, err := renderer.New(rendererConfig, content, fibr.FuncMap(thumbnailApp))
	logger.Fatal(err)

	shareApp := share.New(shareConfig, storageApp)
	crudApp, err := crud.New(crudConfig, storageApp, rendererApp, shareApp, thumbnailApp, exifApp, eventBus.Push)
	logger.Fatal(err)

	var middlewareApp authMiddleware.App
	if !*disableAuth {
		middlewareApp = newLoginApp(basicConfig)
	}

	fibrApp := fibr.New(crudApp, rendererApp, shareApp, middlewareApp)
	handler := rendererApp.Handler(fibrApp.TemplateFunc)

	go shareApp.Start(healthApp.Done())
	go crudApp.Start(healthApp.Done())
	go exifApp.Start(healthApp.Done())
	go eventBus.Start(healthApp.Done(), thumbnailApp.EventConsumer, exifApp.EventConsumer)

	go promServer.Start("prometheus", healthApp.End(), prometheusApp.Handler())
	go appServer.Start("http", healthApp.End(), httputils.Handler(handler, healthApp, recoverer.Middleware, prometheusApp.Middleware, owasp.New(owaspConfig).Middleware))

	healthApp.WaitForTermination(appServer.Done())
	server.GracefulWait(appServer.Done(), promServer.Done())
}
