package main

import (
	"flag"
	"os"
	"time"

	"github.com/ViBiOh/absto/pkg/absto"
	basicMemory "github.com/ViBiOh/auth/v3/pkg/store/memory"
	"github.com/ViBiOh/fibr/pkg/crud"
	"github.com/ViBiOh/fibr/pkg/metadata"
	"github.com/ViBiOh/fibr/pkg/push"
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
	"github.com/ViBiOh/httputils/v4/pkg/pprof"
	"github.com/ViBiOh/httputils/v4/pkg/redis"
	"github.com/ViBiOh/httputils/v4/pkg/renderer"
	"github.com/ViBiOh/httputils/v4/pkg/server"
	"github.com/ViBiOh/httputils/v4/pkg/telemetry"
)

type configuration struct {
	logger    *logger.Config
	alcotest  *alcotest.Config
	telemetry *telemetry.Config
	pprof     *pprof.Config
	health    *health.Config

	server   *server.Config
	owasp    *owasp.Config
	renderer *renderer.Config

	basic         *basicMemory.Config
	absto         *absto.Config
	storage       *storage.Config
	redis         *redis.Config
	amqp          *amqp.Config
	amqpThumbnail *amqphandler.Config
	amqpExif      *amqphandler.Config

	crud      *crud.Config
	sanitizer *sanitizer.Config
	metadata  *metadata.Config
	thumbnail *thumbnail.Config
	webhook   *webhook.Config
	share     *share.Config
	push      *push.Config

	disableAuth           bool
	disableStorageTracing bool
}

func newConfig() configuration {
	fs := flag.NewFlagSet("fibr", flag.ExitOnError)
	fs.Usage = flags.Usage(fs)

	config := configuration{
		logger:    logger.Flags(fs, "logger"),
		alcotest:  alcotest.Flags(fs, ""),
		telemetry: telemetry.Flags(fs, "telemetry"),
		pprof:     pprof.Flags(fs, "pprof"),
		health:    health.Flags(fs, ""),

		server:   server.Flags(fs, "", flags.NewOverride("ReadTimeout", time.Minute*10), flags.NewOverride("WriteTimeout", time.Minute*10)),
		owasp:    owasp.Flags(fs, "", flags.NewOverride("FrameOptions", "SAMEORIGIN"), flags.NewOverride("Csp", "default-src 'self'; base-uri 'self'; script-src 'self' 'httputils-nonce' unpkg.com/webp-hero@0.0.2/dist-cjs/ unpkg.com/leaflet@1.9.4/dist/ unpkg.com/leaflet.markercluster@1.5.1/ cdn.jsdelivr.net/npm/pdfjs-dist@5.4.296/; style-src 'self' 'httputils-nonce' unpkg.com/leaflet@1.9.4/dist/ unpkg.com/leaflet.markercluster@1.5.1/; img-src 'self' data: a.tile.openstreetmap.org b.tile.openstreetmap.org c.tile.openstreetmap.org; worker-src 'self' blob:")),
		renderer: renderer.Flags(fs, "", flags.NewOverride("PublicURL", "http://localhost:1080"), flags.NewOverride("Title", "fibr")),

		basic:         basicMemory.Flags(fs, "auth", flags.NewOverride("Profiles", []string{"1:admin"})),
		absto:         absto.Flags(fs, "storage"),
		storage:       storage.Flags(fs, ""),
		redis:         redis.Flags(fs, "redis", flags.NewOverride("Address", []string{})),
		amqp:          amqp.Flags(fs, "amqp"),
		amqpThumbnail: amqphandler.Flags(fs, "amqpThumbnail", flags.NewOverride("Exchange", "fibr"), flags.NewOverride("Queue", "fibr.thumbnail"), flags.NewOverride("RoutingKey", "thumbnail_output")),
		amqpExif:      amqphandler.Flags(fs, "amqpExif", flags.NewOverride("Exchange", "fibr"), flags.NewOverride("Queue", "fibr.exif"), flags.NewOverride("RoutingKey", "exif_output")),

		crud:      crud.Flags(fs, ""),
		sanitizer: sanitizer.Flags(fs, ""),
		metadata:  metadata.Flags(fs, "exif"),
		thumbnail: thumbnail.Flags(fs, "thumbnail"),
		webhook:   webhook.Flags(fs, "webhook"),
		share:     share.Flags(fs, "share"),
		push:      push.Flags(fs, "push"),
	}

	flags.New("NoAuth", "Disable basic authentification").DocPrefix("auth").BoolVar(fs, &config.disableAuth, false, nil)
	flags.New("NoStorageTrace", "Disable tracing for storage").DocPrefix("storage").BoolVar(fs, &config.disableStorageTracing, false, nil)

	_ = fs.Parse(os.Args[1:])

	return config
}
