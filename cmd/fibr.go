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
	httputils "github.com/ViBiOh/httputils/v3/pkg"
	"github.com/ViBiOh/httputils/v3/pkg/alcotest"
	"github.com/ViBiOh/httputils/v3/pkg/logger"
	"github.com/ViBiOh/httputils/v3/pkg/owasp"
	"github.com/ViBiOh/httputils/v3/pkg/prometheus"
)

func main() {
	fs := flag.NewFlagSet("fibr", flag.ExitOnError)

	serverConfig := httputils.Flags(fs, "")
	alcotestConfig := alcotest.Flags(fs, "")
	prometheusConfig := prometheus.Flags(fs, "prometheus")
	owaspConfig := owasp.Flags(fs, "")

	authConfig := auth.Flags(fs, "auth")
	basicConfig := basic.Flags(fs, "basic")
	crudConfig := crud.Flags(fs, "")
	rendererConfig := renderer.Flags(fs, "")

	filesystemConfig := filesystem.Flags(fs, "fs")
	thumbnailConfig := thumbnail.Flags(fs, "thumbnail")

	logger.Fatal(fs.Parse(os.Args[1:]))

	alcotest.DoAndExit(alcotestConfig)

	prometheusApp := prometheus.New(prometheusConfig)
	owaspApp := owasp.New(owaspConfig)

	storage, err := filesystem.New(filesystemConfig)
	logger.Fatal(err)

	thumbnailApp := thumbnail.New(thumbnailConfig, storage)
	rendererApp := renderer.New(rendererConfig, storage.Root(), thumbnailApp)
	crudApp := crud.New(crudConfig, storage, rendererApp, thumbnailApp)
	authApp := auth.NewService(authConfig, authService.NewBasic(basicConfig, nil))
	fibrApp := fibr.New(crudApp, rendererApp, authApp)

	webHandler := httputils.ChainMiddlewares(fibrApp.Handler(), prometheusApp, owaspApp)

	httputils.New(serverConfig).ListenAndServe(webHandler, httputils.HealthHandler(nil), nil)
}
