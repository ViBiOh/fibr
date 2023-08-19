package main

import (
	"flag"
	"os"
	"time"

	"github.com/ViBiOh/absto/pkg/absto"
	basicMemory "github.com/ViBiOh/auth/v2/pkg/store/memory"
	"github.com/ViBiOh/fibr/pkg/crud"
	"github.com/ViBiOh/fibr/pkg/metadata"
	"github.com/ViBiOh/fibr/pkg/sanitizer"
	"github.com/ViBiOh/fibr/pkg/share"
	"github.com/ViBiOh/fibr/pkg/storage"
	"github.com/ViBiOh/fibr/pkg/thumbnail"
	"github.com/ViBiOh/fibr/pkg/webhook"
	"github.com/ViBiOh/flags"
	"github.com/ViBiOh/httputils/v4/pkg/alcotest"
	"github.com/ViBiOh/httputils/v4/pkg/amqp"
	"github.com/ViBiOh/httputils/v4/pkg/amqphandler"
	"github.com/ViBiOh/httputils/v4/pkg/health"
	"github.com/ViBiOh/httputils/v4/pkg/logger"
	"github.com/ViBiOh/httputils/v4/pkg/owasp"
	"github.com/ViBiOh/httputils/v4/pkg/redis"
	"github.com/ViBiOh/httputils/v4/pkg/renderer"
	"github.com/ViBiOh/httputils/v4/pkg/server"
	"github.com/ViBiOh/httputils/v4/pkg/telemetry"
)

type configuration struct {
	alcotest      alcotest.Config
	logger        logger.Config
	telemetry     telemetry.Config
	appServer     server.Config
	health        health.Config
	owasp         owasp.Config
	basic         basicMemory.Config
	storage       storage.Config
	crud          crud.Config
	sanitizer     sanitizer.Config
	share         share.Config
	webhook       webhook.Config
	renderer      renderer.Config
	absto         absto.Config
	thumbnail     thumbnail.Config
	metadata      metadata.Config
	amqp          amqp.Config
	amqpThumbnail amqphandler.Config
	amqpExif      amqphandler.Config
	redis         redis.Config
	disableAuth   *bool
}

func newConfig() (configuration, error) {
	fs := flag.NewFlagSet("fibr", flag.ExitOnError)
	fs.Usage = flags.Usage(fs)

	return configuration{
		appServer:     server.Flags(fs, "", flags.NewOverride("ReadTimeout", time.Minute*2), flags.NewOverride("WriteTimeout", time.Minute*2)),
		health:        health.Flags(fs, ""),
		alcotest:      alcotest.Flags(fs, ""),
		logger:        logger.Flags(fs, "logger"),
		telemetry:     telemetry.Flags(fs, "telemetry"),
		owasp:         owasp.Flags(fs, "", flags.NewOverride("FrameOptions", "SAMEORIGIN"), flags.NewOverride("Csp", "default-src 'self'; base-uri 'self'; script-src 'self' 'httputils-nonce' unpkg.com/webp-hero@0.0.2/dist-cjs/ unpkg.com/leaflet@1.9.4/dist/ unpkg.com/leaflet.markercluster@1.5.1/; style-src 'self' 'httputils-nonce' unpkg.com/leaflet@1.9.4/dist/ unpkg.com/leaflet.markercluster@1.5.1/; img-src 'self' data: a.tile.openstreetmap.org b.tile.openstreetmap.org c.tile.openstreetmap.org")),
		basic:         basicMemory.Flags(fs, "auth", flags.NewOverride("Profiles", "1:admin")),
		storage:       storage.Flags(fs, ""),
		crud:          crud.Flags(fs, ""),
		sanitizer:     sanitizer.Flags(fs, ""),
		share:         share.Flags(fs, "share"),
		webhook:       webhook.Flags(fs, "webhook"),
		renderer:      renderer.Flags(fs, "", flags.NewOverride("PublicURL", "http://localhost:1080"), flags.NewOverride("Title", "fibr")),
		absto:         absto.Flags(fs, "storage"),
		thumbnail:     thumbnail.Flags(fs, "thumbnail"),
		metadata:      metadata.Flags(fs, "exif"),
		amqp:          amqp.Flags(fs, "amqp"),
		amqpThumbnail: amqphandler.Flags(fs, "amqpThumbnail", flags.NewOverride("Exchange", "fibr"), flags.NewOverride("Queue", "fibr.thumbnail"), flags.NewOverride("RoutingKey", "thumbnail_output")),
		amqpExif:      amqphandler.Flags(fs, "amqpExif", flags.NewOverride("Exchange", "fibr"), flags.NewOverride("Queue", "fibr.exif"), flags.NewOverride("RoutingKey", "exif_output")),
		redis:         redis.Flags(fs, "redis", flags.NewOverride("Address", []string{})),
		disableAuth:   flags.New("NoAuth", "Disable basic authentification").DocPrefix("auth").Bool(fs, false, nil),
	}, fs.Parse(os.Args[1:])
}
