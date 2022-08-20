package main

import (
	"crypto/rand"
	"embed"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"time"

	_ "net/http/pprof"

	"github.com/ViBiOh/absto/pkg/absto"
	"github.com/ViBiOh/auth/v2/pkg/ident/basic"
	authMiddleware "github.com/ViBiOh/auth/v2/pkg/middleware"
	basicMemory "github.com/ViBiOh/auth/v2/pkg/store/memory"
	"github.com/ViBiOh/fibr/pkg/crud"
	"github.com/ViBiOh/fibr/pkg/exif"
	"github.com/ViBiOh/fibr/pkg/fibr"
	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/fibr/pkg/share"
	"github.com/ViBiOh/fibr/pkg/thumbnail"
	"github.com/ViBiOh/fibr/pkg/webhook"
	"github.com/ViBiOh/flags"
	"github.com/ViBiOh/httputils/v4/pkg/alcotest"
	"github.com/ViBiOh/httputils/v4/pkg/amqp"
	"github.com/ViBiOh/httputils/v4/pkg/amqphandler"
	"github.com/ViBiOh/httputils/v4/pkg/health"
	"github.com/ViBiOh/httputils/v4/pkg/httputils"
	"github.com/ViBiOh/httputils/v4/pkg/logger"
	"github.com/ViBiOh/httputils/v4/pkg/owasp"
	"github.com/ViBiOh/httputils/v4/pkg/prometheus"
	"github.com/ViBiOh/httputils/v4/pkg/recoverer"
	"github.com/ViBiOh/httputils/v4/pkg/redis"
	"github.com/ViBiOh/httputils/v4/pkg/renderer"
	"github.com/ViBiOh/httputils/v4/pkg/request"
	"github.com/ViBiOh/httputils/v4/pkg/server"
	"github.com/ViBiOh/httputils/v4/pkg/tracer"
)

//go:embed templates static
var content embed.FS

func newLoginApp(tracerApp tracer.App, basicConfig basicMemory.Config) provider.Auth {
	basicApp, err := basicMemory.New(basicConfig)
	logger.Fatal(err)

	basicProviderProvider := basic.New(basicApp, "fibr")
	return authMiddleware.New(basicApp, tracerApp, basicProviderProvider)
}

func generateIdentityName() string {
	raw := make([]byte, 4)
	if _, err := rand.Read(raw); err != nil {
		logger.Error("generate identity name: %s", err)
		return "error"
	}

	return fmt.Sprintf("%x", raw)
}

func main() {
	fs := flag.NewFlagSet("fibr", flag.ExitOnError)

	appServerConfig := server.Flags(fs, "", flags.NewOverride("ReadTimeout", 2*time.Minute), flags.NewOverride("WriteTimeout", 2*time.Minute))
	promServerConfig := server.Flags(fs, "prometheus", flags.NewOverride("Port", uint(9090)), flags.NewOverride("IdleTimeout", 10*time.Second), flags.NewOverride("ShutdownTimeout", 5*time.Second))
	healthConfig := health.Flags(fs, "")

	alcotestConfig := alcotest.Flags(fs, "")
	loggerConfig := logger.Flags(fs, "logger")
	tracerConfig := tracer.Flags(fs, "tracer")
	prometheusConfig := prometheus.Flags(fs, "prometheus", flags.NewOverride("Gzip", false))
	owaspConfig := owasp.Flags(fs, "", flags.NewOverride("FrameOptions", "SAMEORIGIN"), flags.NewOverride("Csp", "default-src 'self'; base-uri 'self'; script-src 'self' 'httputils-nonce' unpkg.com/webp-hero@0.0.2/dist-cjs/ unpkg.com/leaflet@1.8.0/dist/ unpkg.com/leaflet.markercluster@1.5.1/; style-src 'httputils-nonce' unpkg.com/leaflet@1.8.0/dist/ unpkg.com/leaflet.markercluster@1.5.1/; img-src 'self' data: a.tile.openstreetmap.org b.tile.openstreetmap.org c.tile.openstreetmap.org"))

	basicConfig := basicMemory.Flags(fs, "auth", flags.NewOverride("Profiles", "1:admin"))

	crudConfig := crud.Flags(fs, "")
	shareConfig := share.Flags(fs, "share")
	webhookConfig := webhook.Flags(fs, "webhook")
	rendererConfig := renderer.Flags(fs, "", flags.NewOverride("PublicURL", "http://localhost:1080"), flags.NewOverride("Title", "fibr"))

	abstoConfig := absto.Flags(fs, "storage")
	thumbnailConfig := thumbnail.Flags(fs, "thumbnail")
	exifConfig := exif.Flags(fs, "exif")

	amqpConfig := amqp.Flags(fs, "amqp")
	amqpExifConfig := amqphandler.Flags(fs, "amqpExif", flags.NewOverride("Exchange", "fibr"), flags.NewOverride("Queue", "fibr.exif"), flags.NewOverride("RoutingKey", "exif_output"))
	amqpShareConfig := amqphandler.Flags(fs, "amqpShare", flags.NewOverride("Exchange", "fibr.shares"), flags.NewOverride("Queue", "fibr.share-"+generateIdentityName()), flags.NewOverride("RoutingKey", "share"), flags.NewOverride("Exclusive", true), flags.NewOverride("RetryInterval", time.Duration(0)))
	amqpWebhookConfig := amqphandler.Flags(fs, "amqpWebhook", flags.NewOverride("Exchange", "fibr.webhooks"), flags.NewOverride("Queue", "fibr.webhook-"+generateIdentityName()), flags.NewOverride("RoutingKey", "webhook"), flags.NewOverride("Exclusive", true), flags.NewOverride("RetryInterval", time.Duration(0)))

	redisConfig := redis.Flags(fs, "redis", flags.NewOverride("Address", ""))

	disableAuth := flags.Bool(fs, "", "auth", "NoAuth", "Disable basic authentification", false, nil)

	logger.Fatal(fs.Parse(os.Args[1:]))

	alcotest.DoAndExit(alcotestConfig)
	logger.Global(logger.New(loggerConfig))
	defer logger.Close()

	tracerApp, err := tracer.New(tracerConfig)
	logger.Fatal(err)
	defer tracerApp.Close()
	request.AddTracerToDefaultClient(tracerApp.GetProvider())

	go func() {
		fmt.Println(http.ListenAndServe("localhost:9999", http.DefaultServeMux))
	}()

	appServer := server.New(appServerConfig)
	promServer := server.New(promServerConfig)
	prometheusApp := prometheus.New(prometheusConfig)
	healthApp := health.New(healthConfig)

	prometheusRegisterer := prometheusApp.Registerer()

	storageProvider, err := absto.New(abstoConfig, tracerApp.GetTracer("storage"))
	logger.Fatal(err)

	eventBus, err := provider.NewEventBus(10, prometheusRegisterer, tracerApp.GetTracer("bus"))
	logger.Fatal(err)

	amqpClient, err := amqp.New(amqpConfig, prometheusApp.Registerer())
	if err != nil && !errors.Is(err, amqp.ErrNoConfig) {
		logger.Fatal(err)
	} else if amqpClient != nil {
		defer amqpClient.Close()
	}

	redisClient := redis.New(redisConfig, prometheusApp.Registerer(), tracerApp.GetTracer("redis"))

	thumbnailApp, err := thumbnail.New(thumbnailConfig, storageProvider, prometheusRegisterer, amqpClient)
	logger.Fatal(err)

	rendererApp, err := renderer.New(rendererConfig, content, fibr.FuncMap, tracerApp.GetTracer("renderer"))
	logger.Fatal(err)

	exifApp, err := exif.New(exifConfig, storageProvider, prometheusRegisterer, tracerApp.GetTracer("exif"), amqpClient, redisClient)
	logger.Fatal(err)

	webhookApp, err := webhook.New(webhookConfig, storageProvider, prometheusRegisterer, amqpClient, rendererApp, thumbnailApp)
	logger.Fatal(err)

	shareApp, err := share.New(shareConfig, storageProvider, amqpClient)
	logger.Fatal(err)

	amqpExifApp, err := amqphandler.New(amqpExifConfig, amqpClient, exifApp.AMQPHandler)
	logger.Fatal(err)

	amqpShareApp, err := amqphandler.New(amqpShareConfig, amqpClient, shareApp.AMQPHandler)
	logger.Fatal(err)

	amqpWebhookApp, err := amqphandler.New(amqpWebhookConfig, amqpClient, webhookApp.AMQPHandler)
	logger.Fatal(err)

	crudApp, err := crud.New(crudConfig, storageProvider, rendererApp, shareApp, webhookApp, thumbnailApp, exifApp, eventBus.Push, amqpClient, tracerApp.GetTracer("crud"))
	logger.Fatal(err)

	var middlewareApp provider.Auth
	if !*disableAuth {
		middlewareApp = newLoginApp(tracerApp, basicConfig)
	}

	fibrApp := fibr.New(&crudApp, rendererApp, shareApp, webhookApp, middlewareApp)
	handler := rendererApp.Handler(fibrApp.TemplateFunc)

	go amqpExifApp.Start(healthApp.Done())
	go amqpShareApp.Start(healthApp.Done())
	go amqpWebhookApp.Start(healthApp.Done())
	go webhookApp.Start(healthApp.Done())
	go shareApp.Start(healthApp.Done())
	go crudApp.Start(healthApp.Done())
	go eventBus.Start(healthApp.Done(), storageProvider, []provider.Renamer{thumbnailApp.Rename, exifApp.Rename}, shareApp.EventConsumer, thumbnailApp.EventConsumer, exifApp.EventConsumer, webhookApp.EventConsumer)

	go promServer.Start("prometheus", healthApp.End(), prometheusApp.Handler())
	go appServer.Start("http", healthApp.End(), httputils.Handler(handler, healthApp, recoverer.Middleware, prometheusApp.Middleware, tracerApp.Middleware, owasp.New(owaspConfig).Middleware))

	healthApp.WaitForTermination(appServer.Done())
	server.GracefulWait(appServer.Done(), promServer.Done(), amqpExifApp.Done(), amqpShareApp.Done(), amqpWebhookApp.Done())
}
