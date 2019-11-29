package main

import (
	"flag"
	"os"

	auth "github.com/ViBiOh/auth/v2/pkg/auth/memory"
	"github.com/ViBiOh/auth/v2/pkg/handler"
	"github.com/ViBiOh/auth/v2/pkg/ident/basic"
	basicProvider "github.com/ViBiOh/auth/v2/pkg/ident/basic/memory"
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

func newLoginApp(basicProviderConfig basicProvider.Config, authConfig auth.Config) handler.App {
	basicProviderApp, err := basicProvider.New(basicProviderConfig)
	logger.Fatal(err)
	basicProviderProvider := basic.New(basicProviderApp)

	authApp, err := auth.New(authConfig)
	logger.Fatal(err)

	return handler.New(authApp, basicProviderProvider)
}

func main() {
	fs := flag.NewFlagSet("fibr", flag.ExitOnError)

	serverConfig := httputils.Flags(fs, "")
	alcotestConfig := alcotest.Flags(fs, "")
	prometheusConfig := prometheus.Flags(fs, "prometheus")
	owaspConfig := owasp.Flags(fs, "")

	basicProviderConfig := basicProvider.Flags(fs, "ident")
	authConfig := auth.Flags(fs, "auth")

	crudConfig := crud.Flags(fs, "")
	rendererConfig := renderer.Flags(fs, "")

	filesystemConfig := filesystem.Flags(fs, "fs")
	thumbnailConfig := thumbnail.Flags(fs, "thumbnail")

	logger.Fatal(fs.Parse(os.Args[1:]))

	alcotest.DoAndExit(alcotestConfig)

	storage, err := filesystem.New(filesystemConfig)
	logger.Fatal(err)

	loginApp := newLoginApp(basicProviderConfig, authConfig)

	thumbnailApp := thumbnail.New(thumbnailConfig, storage)
	rendererApp := renderer.New(rendererConfig, storage.Root(), thumbnailApp)
	crudApp := crud.New(crudConfig, storage, rendererApp, thumbnailApp)
	fibrApp := fibr.New(crudApp, rendererApp, loginApp)

	server := httputils.New(serverConfig)
	server.Middleware(prometheus.New(prometheusConfig))
	server.Middleware(owasp.New(owaspConfig))
	server.ListenServeWait(fibrApp.Handler())
}
