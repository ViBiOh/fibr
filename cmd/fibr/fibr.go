package main

import (
	"flag"
	"os"

	"github.com/ViBiOh/auth/v2/pkg/handler"
	"github.com/ViBiOh/auth/v2/pkg/ident/basic"
	basicMemory "github.com/ViBiOh/auth/v2/pkg/provider/memory"
	"github.com/ViBiOh/fibr/pkg/crud"
	"github.com/ViBiOh/fibr/pkg/fibr"
	"github.com/ViBiOh/fibr/pkg/filesystem"
	"github.com/ViBiOh/fibr/pkg/renderer"
	"github.com/ViBiOh/fibr/pkg/thumbnail"
	"github.com/ViBiOh/httputils/v3/pkg/alcotest"
	"github.com/ViBiOh/httputils/v3/pkg/httputils"
	"github.com/ViBiOh/httputils/v3/pkg/logger"
	"github.com/ViBiOh/httputils/v3/pkg/owasp"
	"github.com/ViBiOh/httputils/v3/pkg/prometheus"
)

func newLoginApp(basicConfig basicMemory.Config) handler.App {
	basicApp, err := basicMemory.New(basicConfig)
	logger.Fatal(err)

	basicProviderProvider := basic.New(basicApp)
	return handler.New(basicApp, basicProviderProvider)
}

func main() {
	fs := flag.NewFlagSet("fibr", flag.ExitOnError)

	serverConfig := httputils.Flags(fs, "")
	alcotestConfig := alcotest.Flags(fs, "")
	prometheusConfig := prometheus.Flags(fs, "prometheus")
	owaspConfig := owasp.Flags(fs, "")

	basicConfig := basicMemory.Flags(fs, "auth")

	crudConfig := crud.Flags(fs, "")
	rendererConfig := renderer.Flags(fs, "")

	filesystemConfig := filesystem.Flags(fs, "fs")
	thumbnailConfig := thumbnail.Flags(fs, "thumbnail")

	logger.Fatal(fs.Parse(os.Args[1:]))

	alcotest.DoAndExit(alcotestConfig)

	storage, err := filesystem.New(filesystemConfig)
	logger.Fatal(err)

	loginApp := newLoginApp(basicConfig)

	thumbnailApp := thumbnail.New(thumbnailConfig, storage)
	rendererApp := renderer.New(rendererConfig, thumbnailApp)
	crudApp, err := crud.New(crudConfig, storage, rendererApp, thumbnailApp)
	logger.Fatal(err)

	fibrApp := fibr.New(crudApp, rendererApp, loginApp)

	go thumbnailApp.Start()
	go crudApp.Start()

	server := httputils.New(serverConfig)
	server.Middleware(prometheus.New(prometheusConfig).Middleware)
	server.Middleware(owasp.New(owaspConfig).Middleware)
	server.ListenServeWait(fibrApp.Handler())
}
