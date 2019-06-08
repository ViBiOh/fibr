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
	"github.com/ViBiOh/httputils/pkg/healthcheck"
	"github.com/ViBiOh/httputils/pkg/logger"
	"github.com/ViBiOh/httputils/pkg/opentracing"
	"github.com/ViBiOh/httputils/pkg/owasp"
	"github.com/ViBiOh/httputils/pkg/prometheus"
	"github.com/ViBiOh/httputils/pkg/server"
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

	if err := fs.Parse(os.Args[1:]); err != nil {
		logger.Fatal("%#v", err)
	}

	alcotest.DoAndExit(alcotestConfig)

	storage, err := filesystem.New(filesystemConfig)
	if err != nil {
		logger.Error("%#v", err)
		os.Exit(1)
	}

	serverApp, err := httputils.New(serverConfig)
	if err != nil {
		logger.Fatal("%#v", err)
	}

	healthcheckApp := healthcheck.New()
	prometheusApp := prometheus.New(prometheusConfig)
	opentracingApp := opentracing.New(opentracingConfig)
	owaspApp := owasp.New(owaspConfig)
	gzipApp := gzip.New()

	thumbnailApp := thumbnail.New(thumbnailConfig, storage)
	rendererApp := renderer.New(rendererConfig, storage.Root(), thumbnailApp)
	crudApp := crud.New(crudConfig, storage, rendererApp, thumbnailApp)
	authApp := auth.NewService(authConfig, authService.NewBasic(basicConfig, nil))
	fibrApp := fibr.New(crudApp, rendererApp, authApp)

	webHandler := server.ChainMiddlewares(fibrApp.Handler(), prometheusApp, opentracingApp, gzipApp, owaspApp)

	serverApp.ListenAndServe(webHandler, nil, healthcheckApp)
}
