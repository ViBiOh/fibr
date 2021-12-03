package main

import (
	"crypto/rand"
	"embed"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"

	_ "net/http/pprof"

	"github.com/ViBiOh/auth/v2/pkg/ident/basic"
	authMiddleware "github.com/ViBiOh/auth/v2/pkg/middleware"
	basicMemory "github.com/ViBiOh/auth/v2/pkg/store/memory"
	"github.com/ViBiOh/fibr/pkg/crud"
	"github.com/ViBiOh/fibr/pkg/exif"
	"github.com/ViBiOh/fibr/pkg/fibr"
	"github.com/ViBiOh/fibr/pkg/filesystem"
	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/fibr/pkg/s3"
	"github.com/ViBiOh/fibr/pkg/share"
	"github.com/ViBiOh/fibr/pkg/thumbnail"
	"github.com/ViBiOh/fibr/pkg/webhook"
	"github.com/ViBiOh/httputils/v4/pkg/alcotest"
	"github.com/ViBiOh/httputils/v4/pkg/amqp"
	"github.com/ViBiOh/httputils/v4/pkg/amqphandler"
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

func newLoginApp(basicConfig basicMemory.Config) provider.Auth {
	basicApp, err := basicMemory.New(basicConfig)
	logger.Fatal(err)

	basicProviderProvider := basic.New(basicApp, "fibr")
	return authMiddleware.New(basicApp, basicProviderProvider)
}

func generateIdentityName() string {
	raw := make([]byte, 4)
	if _, err := rand.Read(raw); err != nil {
		logger.Error("unable to generate identity name: %s", err)
		return "error"
	}

	return fmt.Sprintf("%x", raw)
}

func main() {
	fs := flag.NewFlagSet("fibr", flag.ExitOnError)

	appServerConfig := server.Flags(fs, "", flags.NewOverride("ReadTimeout", "2m"), flags.NewOverride("WriteTimeout", "2m"))
	promServerConfig := server.Flags(fs, "prometheus", flags.NewOverride("Port", 9090), flags.NewOverride("IdleTimeout", "10s"), flags.NewOverride("ShutdownTimeout", "5s"))
	healthConfig := health.Flags(fs, "")

	alcotestConfig := alcotest.Flags(fs, "")
	loggerConfig := logger.Flags(fs, "logger")
	prometheusConfig := prometheus.Flags(fs, "prometheus", flags.NewOverride("Gzip", false))
	owaspConfig := owasp.Flags(fs, "", flags.NewOverride("FrameOptions", "SAMEORIGIN"), flags.NewOverride("Csp", "default-src 'self'; base-uri 'self'; script-src 'nonce'; style-src 'nonce'; img-src 'self' data:"))

	basicConfig := basicMemory.Flags(fs, "auth")

	crudConfig := crud.Flags(fs, "")
	shareConfig := share.Flags(fs, "share")
	webhookConfig := webhook.Flags(fs, "webhook")
	rendererConfig := renderer.Flags(fs, "", flags.NewOverride("PublicURL", "https://fibr.vibioh.fr"), flags.NewOverride("Title", "fibr"))

	filesystemConfig := filesystem.Flags(fs, "fs")
	s3Config := s3.Flags(fs, "s3")
	thumbnailConfig := thumbnail.Flags(fs, "thumbnail")
	exifConfig := exif.Flags(fs, "exif")

	amqpConfig := amqp.Flags(fs, "amqp")
	amqpExifConfig := amqphandler.Flags(fs, "amqpExif", flags.NewOverride("Exchange", "fibr"), flags.NewOverride("Queue", "fibr.exif"), flags.NewOverride("RoutingKey", "exif"))
	amqpShareConfig := amqphandler.Flags(fs, "amqpShare", flags.NewOverride("Exchange", "fibr.shares"), flags.NewOverride("Queue", "fibr.share-"+generateIdentityName()), flags.NewOverride("RoutingKey", "share"), flags.NewOverride("Exclusive", true), flags.NewOverride("RetryInterval", "0"))
	amqpWebhookConfig := amqphandler.Flags(fs, "amqpWebhook", flags.NewOverride("Exchange", "fibr.webhooks"), flags.NewOverride("Queue", "fibr.webhook-"+generateIdentityName()), flags.NewOverride("RoutingKey", "webhook"), flags.NewOverride("Exclusive", true), flags.NewOverride("RetryInterval", "0"))

	disableAuth := flags.New("", "auth", "NoAuth").Default(false, nil).Label("Disable basic authentification").ToBool(fs)

	logger.Fatal(fs.Parse(os.Args[1:]))

	alcotest.DoAndExit(alcotestConfig)
	logger.Global(logger.New(loggerConfig))
	defer logger.Close()

	go func() {
		fmt.Println(http.ListenAndServe("localhost:9999", http.DefaultServeMux))
	}()

	appServer := server.New(appServerConfig)
	promServer := server.New(promServerConfig)
	prometheusApp := prometheus.New(prometheusConfig)
	healthApp := health.New(healthConfig)

	s3App, err := s3.New(s3Config)
	logger.Fatal(err)

	var storageProvider provider.Storage

	if s3App.Enabled() {
		logger.Info("Serving content from s3")
		storageProvider = s3App
	} else {
		fsApp, err := filesystem.New(filesystemConfig)
		logger.Fatal(err)

		logger.Info("Serving content from filesystem")
		storageProvider = fsApp
	}

	prometheusRegisterer := prometheusApp.Registerer()
	eventBus, err := provider.NewEventBus(10, prometheusRegisterer)
	logger.Fatal(err)

	amqpClient, err := amqp.New(amqpConfig, prometheusApp.Registerer())
	if err != nil && !errors.Is(err, amqp.ErrNoConfig) {
		logger.Error("unable to create amqp client: %s", err)
	} else if amqpClient != nil {
		defer amqpClient.Close()
	}

	thumbnailApp, err := thumbnail.New(thumbnailConfig, storageProvider, prometheusRegisterer, amqpClient)
	logger.Fatal(err)

	exifApp, err := exif.New(exifConfig, storageProvider, prometheusRegisterer, amqpClient)
	logger.Fatal(err)

	webhookApp, err := webhook.New(webhookConfig, storageProvider, prometheusRegisterer, amqpClient)
	logger.Fatal(err)

	rendererApp, err := renderer.New(rendererConfig, content, fibr.FuncMap(thumbnailApp))
	logger.Fatal(err)

	shareApp, err := share.New(shareConfig, storageProvider, amqpClient)
	logger.Fatal(err)

	amqpExifApp, err := amqphandler.New(amqpExifConfig, amqpClient, exifApp.AmqpHandler)
	logger.Fatal(err)

	amqpShareApp, err := amqphandler.New(amqpShareConfig, amqpClient, shareApp.AmqpHandler)
	logger.Fatal(err)

	amqpWebhookApp, err := amqphandler.New(amqpWebhookConfig, amqpClient, webhookApp.AmqpHandler)
	logger.Fatal(err)

	crudApp, err := crud.New(crudConfig, storageProvider, rendererApp, shareApp, webhookApp, thumbnailApp, exifApp, eventBus.Push, amqpClient)
	logger.Fatal(err)

	var middlewareApp provider.Auth
	if !*disableAuth {
		middlewareApp = newLoginApp(basicConfig)
	}

	fibrApp := fibr.New(&crudApp, rendererApp, shareApp, webhookApp, middlewareApp)
	handler := rendererApp.Handler(fibrApp.TemplateFunc)

	go amqpExifApp.Start(healthApp.Done())
	go amqpShareApp.Start(healthApp.Done())
	go amqpWebhookApp.Start(healthApp.Done())
	go webhookApp.Start(healthApp.Done())
	go shareApp.Start(healthApp.Done())
	go crudApp.Start(healthApp.Done())
	go eventBus.Start(healthApp.Done(), shareApp.EventConsumer, thumbnailApp.EventConsumer, webhookApp.EventConsumer, exifApp.EventConsumer)

	go promServer.Start("prometheus", healthApp.End(), prometheusApp.Handler())
	go appServer.Start("http", healthApp.End(), httputils.Handler(handler, healthApp, recoverer.Middleware, prometheusApp.Middleware, owasp.New(owaspConfig).Middleware))

	healthApp.WaitForTermination(appServer.Done())
	server.GracefulWait(appServer.Done(), promServer.Done(), amqpExifApp.Done(), amqpShareApp.Done(), amqpWebhookApp.Done())
}
