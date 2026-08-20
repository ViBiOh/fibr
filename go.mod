module github.com/ViBiOh/fibr

go 1.26.0

require (
	codeberg.org/ViBiOh/ChatPotte v0.11.0
	github.com/ViBiOh/absto v1.7.35
	github.com/ViBiOh/auth/v3 v3.11.1
	github.com/ViBiOh/exas v0.8.1
	github.com/ViBiOh/flags v1.6.1
	github.com/ViBiOh/httputils/v4 v4.88.2
	github.com/ViBiOh/vignet v0.0.1
	github.com/golang-jwt/jwt/v5 v5.3.1
	github.com/rabbitmq/amqp091-go v1.13.0
	github.com/redis/go-redis/v9 v9.22.0
	github.com/rs/xid v1.6.0
	github.com/zeebo/xxh3 v1.1.0
	go.opentelemetry.io/otel v1.45.0
	go.opentelemetry.io/otel/metric v1.45.0
	go.opentelemetry.io/otel/trace v1.45.0
	go.uber.org/mock v0.6.0
	golang.org/x/crypto v0.55.0
	golang.org/x/text v0.41.0
)

require (
	github.com/cenkalti/backoff/v5 v5.0.3 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/felixge/httpsnoop v1.1.0 // indirect
	github.com/go-logr/logr v1.4.4 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.30.0 // indirect
	github.com/klauspost/compress v1.19.0 // indirect
	github.com/klauspost/cpuid/v2 v2.4.0 // indirect
	github.com/klauspost/crc32 v1.3.0 // indirect
	github.com/minio/crc64nvme v1.1.1 // indirect
	github.com/minio/md5-simd v1.1.2 // indirect
	github.com/minio/minio-go/v7 v7.2.1 // indirect
	github.com/philhofer/fwd v1.2.0 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/redis/go-redis/extra/rediscmd/v9 v9.22.0 // indirect
	github.com/redis/go-redis/extra/redisotel/v9 v9.22.0 // indirect
	github.com/tdewolff/minify/v2 v2.24.16 // indirect
	github.com/tdewolff/parse/v2 v2.8.15 // indirect
	github.com/tinylib/msgp v1.6.4 // indirect
	go.opentelemetry.io/auto/sdk v1.2.1 // indirect
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.70.0 // indirect
	go.opentelemetry.io/contrib/instrumentation/runtime v0.70.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc v1.45.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace v1.45.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc v1.45.0 // indirect
	go.opentelemetry.io/otel/sdk v1.45.0 // indirect
	go.opentelemetry.io/otel/sdk/metric v1.45.0 // indirect
	go.opentelemetry.io/proto/otlp v1.11.0 // indirect
	go.uber.org/atomic v1.11.0 // indirect
	go.yaml.in/yaml/v3 v3.0.5 // indirect
	golang.org/x/mod v0.39.0 // indirect
	golang.org/x/net v0.58.0 // indirect
	golang.org/x/oauth2 v0.36.0 // indirect
	golang.org/x/sync v0.22.0 // indirect
	golang.org/x/sys v0.47.0 // indirect
	golang.org/x/telemetry v0.0.0-20260811182544-a038080d80e5 // indirect
	golang.org/x/term v0.45.0 // indirect
	golang.org/x/tools v0.49.1-0.20260819203639-c62e53519fb7 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20260807164820-c8921c73eeea // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260807164820-c8921c73eeea // indirect
	google.golang.org/grpc v1.83.0 // indirect
	google.golang.org/protobuf v1.36.11 // indirect
	gopkg.in/ini.v1 v1.67.3 // indirect
	mvdan.cc/gofumpt v0.11.0 // indirect
)

tool (
	go.uber.org/mock/mockgen
	golang.org/x/tools/cmd/goimports
	golang.org/x/tools/go/analysis/passes/fieldalignment/cmd/fieldalignment
	mvdan.cc/gofumpt
)
