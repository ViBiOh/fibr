package main

import (
	"flag"
	"os"

	"github.com/ViBiOh/auth/pkg/auth"
	"github.com/ViBiOh/auth/pkg/ident/basic"
	authService "github.com/ViBiOh/auth/pkg/ident/service"
	"github.com/ViBiOh/fibr/pkg/crud"
	"github.com/ViBiOh/fibr/pkg/fibr"
	"github.com/ViBiOh/fibr/pkg/filesystem"
	"github.com/ViBiOh/fibr/pkg/renderer"
	"github.com/ViBiOh/fibr/pkg/thumbnail"
	httputils "github.com/ViBiOh/httputils/pkg"
	"github.com/ViBiOh/httputils/pkg/alcotest"
	"github.com/ViBiOh/httputils/pkg/gzip"
	"github.com/ViBiOh/httputils/pkg/logger"
	"github.com/ViBiOh/httputils/pkg/opentracing"
	"github.com/ViBiOh/httputils/pkg/owasp"
	"github.com/ViBiOh/httputils/pkg/prometheus"
)

func main() {
	fs := flag.NewFlagSet("fibr", flag.ExitOnError)

	serverConfig := httputils.Flags(fs, "")
	alcotestConfig := alcotest.Flags(fs, "")
	prometheusConfig := prometheus.Flags(fs, "prometheus")
	opentracingConfig := opentracing.Flags(fs, "tracing")
	owaspConfig := owasp.Flags(fs, "")

	authConfig := auth.Flags(fs, "auth")
	basicConfig := basic.Flags(fs, "basic")
	crudConfig := crud.Flags(fs, "")
	rendererConfig := renderer.Flags(fs, "")

	filesystemConfig := filesystem.Flags(fs, "fs")
	thumbnailConfig := thumbnail.Flags(fs, "thumbnail")

	logger.Fatal(fs.Parse(os.Args[1:]))

	alcotest.DoAndExit(alcotestConfig)

	serverApp, err := httputils.New(serverConfig)
	logger.Fatal(err)

	prometheusApp := prometheus.New(prometheusConfig)
	opentracingApp := opentracing.New(opentracingConfig)
	owaspApp := owasp.New(owaspConfig)
	gzipApp := gzip.New()

	storage, err := filesystem.New(filesystemConfig)
	logger.Fatal(err)

	thumbnailApp := thumbnail.New(thumbnailConfig, storage)
	rendererApp := renderer.New(rendererConfig, storage.Root(), thumbnailApp)
	crudApp := crud.New(crudConfig, storage, rendererApp, thumbnailApp)
	authApp := auth.NewService(authConfig, authService.NewBasic(basicConfig, nil))
	fibrApp := fibr.New(crudApp, rendererApp, authApp)

	webHandler := httputils.ChainMiddlewares(fibrApp.Handler(), prometheusApp, opentracingApp, gzipApp, owaspApp)

	serverApp.ListenAndServe(webHandler, httputils.HealthHandler(nil), nil)
}
