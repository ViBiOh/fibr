# fibr

Web File Browser and Manager.

- Lightweight (13MB self-sufficient binary, low memory consumption at runtime).
- Mobile-first interface, with light payload. Dark themed.
- Thumbnail generation for image, PDF and video (with help of sidecars)
- Exif extraction and displaying content on a map (with help of sidecars)
- Works in pure HTML or with very little javascript for improved file upload
- Support for basic filesystem and storage object (in beta)
- Can share directory with ou without password and with or without edit right.
- Can communicate with sidecars in pure HTTP or AMQP
- Can send webhooks for different event types to various providers
- Basic search for files on metadatas without indexation
- OpenTelemetry and pprof already built-in

![](docs/fibr.png)

[![Build](https://github.com/ViBiOh/fibr/workflows/Build/badge.svg)](https://github.com/ViBiOh/fibr/actions)
[![codecov](https://codecov.io/gh/ViBiOh/fibr/branch/main/graph/badge.svg)](https://codecov.io/gh/ViBiOh/fibr)
[![Quality Gate Status](https://sonarcloud.io/api/project_badges/measure?project=ViBiOh_fibr&metric=alert_status)](https://sonarcloud.io/dashboard?id=ViBiOh_fibr)

Thanks to [FontAwesome](https://fontawesome.com) for providing awesome svg.

I do it mostly for myself, but if you want to support me, you can star this project to give more visibility or sponsor my work (it pays server hosting, tooling, etc.).

[!["Buy Me A Tea"](https://img.buymeacoffee.com/button-api/?text=Buy%20me%20a%20tea&emoji=🍵&slug=vibioh&button_colour=5F7FFF&font_colour=ffffff&font_family=Lato&outline_colour=000000&coffee_colour=FFDD00)](https://www.buymeacoffee.com/vibioh)

## Concepts

Fibr aims to provide simple browsing of your filesystem. It's a single static binary with embedded html templates. No Javascript framework. HTTP and HTML have all we need.

Fibr aims to be compatible with the most platforms available, on a best-effort basis. Fibr itself is already compatible with `x86_64`, `arm`, `arm64` architectures. But sidecars, which depends on system library, are not all ready yet.

### Folder

Fibr browses files of given `-data` option folder (or S3 configuration), called "root folder". For security reason, it's not possible to browse parent.

It aims to be consistent accross all existing filesystem (block storage, object storage, etc.) and thus enforces filenames in lowercase, with no space or special character. At start, it walks every files and reports names that breaks its policy. It doesn't modify existing files unless you set `-sanitizeOnStart` option.

Fibr creates a `.fibr` folder in _root folder_ for storing its metadata: shares' configuration, thumbnails and exif. If you want to stop using _fibr_ or start with a fresh installation (e.g. regenerating thumbnails), you can delete this folder.

### Sidecars

Fibr generates thumbnails of images, PDF and videos when these [mime-types are detected](https://developer.mozilla.org/en-US/docs/Web/HTTP/Basics_of_HTTP/MIME_types/Common_types) and sidecars are provided. Sidecars are [h2non/imaginary](https://github.com/h2non/imaginary), [ViBiOh/vith](https://github.com/vibioh/vith) and [ViBiOh/exas](https://github.com/vibioh/exas). Thumbnails are generated in [WebP](https://developers.google.com/speed/webp/) format, in their animated format for video thumbnail.

You can refer to these projects for installing and configuring them and set `-thumbnailURL` and `-exifURL` options.

Sidecars may have constraints regarding concurrent work (e.g. HLS conversion is a CPU-intensive task) or rate limit (e.g. geocoding can have rate-limiting). Call to these sidecars can be made with HTTP, which is not fault tolerant but easy to setup, or with an AMQP messaging, which is more resilient but more complex to setup. An easy-to-setup AMQP messaging instance can be done with [CloudAMQP](https://www.cloudamqp.com) (I have no affiliation of any kind to this company, just a happy customer). When AMQP connection URI is provided, Fibr will use it as default communication protocol instead of HTTP.

#### HTTP Live Streaming

Fibr has a special treatment for videos, that can be very large sometimes. With the help of the `vith` sidecar, it can convert a video to its [HLS version](https://en.wikipedia.org/wiki/HTTP_Live_Streaming). It keeps the original video as is, and stores streamable version in the metadatas directory. It's a basic conversion into the appropriate format: no resolution, frame-per-second or any quality specifications are changed. Conversion is done where this two requirements are met altogether:

- `vith` is configured with direct access to the filesystem (see [`vith`documentation about configuring `WorkDir`](https://github.com/vibioh/vith#usage) and [`fibr` configuration](#usage) for enabling it). Direct access disable large file transfer in the network.
- the video bitrate is above [`thumbnailMinBitrate (default 80000000)`](#usage)

### Security

Authentication is made with [Basic Auth](https://developer.mozilla.org/en-US/docs/Web/HTTP/Authentication), compatible with all browsers and CLI tools such as `curl`. I **strongly recommend configuring HTTPS** in order to avoid exposing your credentials in plain text.

You can set `-cert` and `-key` options, it uses [Golang's standard net/http#ServeTLS](https://golang.org/pkg/net/http/#ServeTLS) under the hood.

You can also configure a reverse proxy with Let's Encrypt to manage encryption, such as [Traefik](https://docs.traefik.io).

### Sharing

You can share folders or just one file: it generates a short link that gain access to shared object and is considered as "root folder" with no parent escalation.

It can be password-protected: user **has to** enter password to see content (login is not used, you can leave it blank).

It can be read-only or with edit right. With edit-right, user can do anything as you, uploading, deleting, renaming, except generating new shares.

It can be created with expiration duration.

> It's really useful for sharing files with friends. You don't need account at Google, Dropbox, iCloud or a mobile-app: a link and everyone can see and share content!

This is the main reason I've started to develop this app.

### Webhook

You can register webhook listeners on folders and receive an HTTP notification when one of these event occurs:

- `create` occurs when a directory is created
- `upload` occurs when an item is uploaded
- `rename` occurs when an item is renamed
- `delete` occurs when an item is deleted
- `start` occurs when fibr start and do something on an item
- `access` occurs when content is accessed (directory browsing or just one file)

The request sent is a POST with 15s timeout with the given payload structure:

```json
{
  "time": "2021-02-25:12:32.244914+01:00",
  "url": "/eventual_share_id/path/to/payload.json",
  "item": {
    "date": "2021-08-10T19:31:28.952325533Z",
    "name": "payload.json",
    "pathname": "/path/to/payload.json",
    "isDir": false,
    "size": 177
  },
  "type": "upload"
}
```

It will contains an extra key `new` with the same structure of `item` in case of a `rename` event, and a `metadata` map in case of `access` event, that contains a dump of HTTP Header (except `Authorization`).

The webhook can be recursive (all children folders will be notified too) for event choosen.

#### Security

Webhooks can be sent with an [HTTP Signature](https://tools.ietf.org/id/draft-cavage-http-signatures-12.html) if you configure the [`webhookSecret`](#usage). It adds an `Authorization` header to the sent request that serves as an authentification mechanism for the receiver: if the signature is not valid, you should not trust the caller.

I've implemented a [very simple function](https://github.com/ViBiOh/httputils/blob/main/pkg/request/signature.go#L34) you can add to your receiver for checking it.

### SEO

Fibr provides [OpenGraph metadatas](https://ogp.me) to have nice preview of link when shared. These metadatas don't leak any password-protected datas.

![](docs/opengraph.png)

### Users

You can start `fibr` with no user, with the `-noAuth` option. Although available, I don't recommend using it in public Internet. Anybody has access to the _root folder_ for viewing, uploading, deleting or sharing content with anybody.

Users are set with the `-authUsers` option and are in the form `[id]:[login]:[bcrypted password]`.

- `id` is used to add profile to your user
- `login` is the user for Basic Auth prompt
- `bcrypted password` is the password for Basic Auth prompt, [encrypted with `bcrypt`](https://en.wikipedia.org/wiki/Bcrypt)

You can easily encrypt your `login:password` value with [`htpasswd`](https://httpd.apache.org/docs/2.4/programs/htpasswd.html)

```bash
htpasswd -nBb login password
```

In order to work, your user **must have** `admin` profile sets with the `-authProfiles` option.

### Metadatas

With help of different sidecars, Fibr can generate image, video and PDF thumbnails. These sidecars can be self hosted with ease. It can also extract and enrich content displayed by looking at [EXIF Data](https://en.wikipedia.org/wiki/Exif), also with the help of a little sidecar. These behaviours are opt-out (if you remove the `url` of the service, Fibr will do nothing).

For the last mile, Fibr can try to reverse geocoding the GPS data found in EXIF, using [Open Street Map](https://wiki.openstreetmap.org/wiki/Nominatim). Self-hosting this kind of service can be complicated and calling a third-party party with such sensible datas is an opt-in decision.

### Metrics

Fibr exposes a lot of metrics on the [Prometheus endpoint](#endpoints). Common metrics are exposed: Golang statistics, HTTP statuses and response time, AMQP statuses and sidecars/metadatas actions.

## Getting started

### As a binary, without authentification

This is for local purpose with simple and hassle-free sharing in a private network.

```bash
go install github.com/ViBiOh/fibr/cmd/fibr@latest
fibr \
  -noAuth \
  -storageDirectory "$(pwd)" \
  -thumbnailURL "" \
  -exifURL ""
```

### As a single Docker container, with admin/password user

For long-living sharing with password and self-contained app in Docker, with no thumbnail generation or exif, configured with environment variables.

```bash
docker run -d \
  -p 1080:1080/tcp \
  --name fibr \
  -v ${PWD}:/data/ \
  -e FIBR_AUTH_USERS="1:$(htpasswd -nBb login password)" \
  -e FIBR_THUMBNAIL_URL="" \
  -e FIBR_EXIF_URL="" \
  vibioh/fibr
```

### As a docker-compose / k8s stack

For prod-ready run with thumbnails generation of image, PDF and videos, _this is the recommended approach_.

You can inspire yourself from the [docker-compose.yaml](docker-compose.yaml) file I personnaly used. Beware of `-authUsers` option: bcrypted passwords contain dollar sign, which `docker-compose` tries to resolve as a shell variable, [you must escape it](https://docs.docker.com/compose/compose-file/compose-file-v2/#variable-substitution).

```bash
DATA_USER_ID="$(id -u)" DATA_DIR="$(pwd)" BASIC_USERS="1:$(htpasswd -nBb admin password)" docker-compose up
```

You'll find a Kubernetes exemple in the [`infra/`](infra/) folder, using my [`app chart`](https://github.com/ViBiOh/charts/tree/main/app). My personnal k8s runs on `arm64` and thumbnail converters are not yet ready for this architecture, so I use a mix of `helm` and `docker-compose.yaml`.

## Endpoints

- `GET /health`: healthcheck of server, always respond [`okStatus (default 204)`](#usage)
- `GET /ready`: checks external dependencies availability and then respond [`okStatus (default 204)`](#usage) or `503` during [`graceDuration`](#usage) when `SIGTERM` is received
- `GET /version`: value of `VERSION` environment variable
- `GET /metrics`: Prometheus metrics, on a dedicated port [`prometheusPort (default 9090)`](#usage)

## Usage

Fibr can be configured by passing CLI args described below or their equivalent as environment variable. If both the CLI and environment variable are defined, the CLI value is used.

Be careful when using the CLI, if someone list the processes on the system, they will appear in plain-text. I recommend passing secrets by environment variables: it's less easily visible.

```bash
Usage of fibr:
  -address string
        [server] Listen address {FIBR_ADDRESS}
  -amqpExclusiveRoutingKey string
        [crud] AMQP Routing Key for exclusive lock on default exchange {FIBR_AMQP_EXCLUSIVE_ROUTING_KEY} (default "fibr.semaphore.start")
  -amqpExifExchange string
        [amqpExif] Exchange name {FIBR_AMQP_EXIF_EXCHANGE} (default "fibr")
  -amqpExifExclusive
        [amqpExif] Queue exclusive mode (for fanout exchange) {FIBR_AMQP_EXIF_EXCLUSIVE}
  -amqpExifMaxRetry uint
        [amqpExif] Max send retries {FIBR_AMQP_EXIF_MAX_RETRY} (default 3)
  -amqpExifQueue string
        [amqpExif] Queue name {FIBR_AMQP_EXIF_QUEUE} (default "fibr.exif")
  -amqpExifRetryInterval string
        [amqpExif] Interval duration when send fails {FIBR_AMQP_EXIF_RETRY_INTERVAL} (default "1h")
  -amqpExifRoutingKey string
        [amqpExif] RoutingKey name {FIBR_AMQP_EXIF_ROUTING_KEY} (default "exif_output")
  -amqpPrefetch int
        [amqp] Prefetch count for QoS {FIBR_AMQP_PREFETCH} (default 1)
  -amqpShareExchange string
        [amqpShare] Exchange name {FIBR_AMQP_SHARE_EXCHANGE} (default "fibr.shares")
  -amqpShareExclusive
        [amqpShare] Queue exclusive mode (for fanout exchange) {FIBR_AMQP_SHARE_EXCLUSIVE} (default true)
  -amqpShareMaxRetry uint
        [amqpShare] Max send retries {FIBR_AMQP_SHARE_MAX_RETRY} (default 3)
  -amqpShareQueue string
        [amqpShare] Queue name {FIBR_AMQP_SHARE_QUEUE} (default "fibr.share-<random>")
  -amqpShareRetryInterval string
        [amqpShare] Interval duration when send fails {FIBR_AMQP_SHARE_RETRY_INTERVAL} (default "0")
  -amqpShareRoutingKey string
        [amqpShare] RoutingKey name {FIBR_AMQP_SHARE_ROUTING_KEY} (default "share")
  -amqpURI string
        [amqp] Address in the form amqps?://<user>:<password>@<address>:<port>/<vhost> {FIBR_AMQP_URI}
  -amqpWebhookExchange string
        [amqpWebhook] Exchange name {FIBR_AMQP_WEBHOOK_EXCHANGE} (default "fibr.webhooks")
  -amqpWebhookExclusive
        [amqpWebhook] Queue exclusive mode (for fanout exchange) {FIBR_AMQP_WEBHOOK_EXCLUSIVE} (default true)
  -amqpWebhookMaxRetry uint
        [amqpWebhook] Max send retries {FIBR_AMQP_WEBHOOK_MAX_RETRY} (default 3)
  -amqpWebhookQueue string
        [amqpWebhook] Queue name {FIBR_AMQP_WEBHOOK_QUEUE} (default "fibr.webhook-<random>")
  -amqpWebhookRetryInterval string
        [amqpWebhook] Interval duration when send fails {FIBR_AMQP_WEBHOOK_RETRY_INTERVAL} (default "0")
  -amqpWebhookRoutingKey string
        [amqpWebhook] RoutingKey name {FIBR_AMQP_WEBHOOK_ROUTING_KEY} (default "webhook")
  -authProfiles string
        [auth] Users profiles in the form 'id:profile1|profile2,id2:profile1' {FIBR_AUTH_PROFILES} (default "1:admin")
  -authUsers string
        [auth] Users credentials in the form 'id:login:password,id2:login2:password2' {FIBR_AUTH_USERS}
  -bcryptDuration string
        [crud] Wanted bcrypt duration for calculating effective cost {FIBR_BCRYPT_DURATION} (default "0.25s")
  -cert string
        [server] Certificate file {FIBR_CERT}
  -csp string
        [owasp] Content-Security-Policy {FIBR_CSP} (default "default-src 'self'; base-uri 'self'; script-src 'httputils-nonce' unpkg.com/leaflet@1.7.1/dist/ unpkg.com/leaflet.markercluster@1.5.1/; style-src 'httputils-nonce' unpkg.com/leaflet@1.7.1/dist/ unpkg.com/leaflet.markercluster@1.5.1/; img-src 'self' data: a.tile.openstreetmap.org b.tile.openstreetmap.org c.tile.openstreetmap.org")
  -exifAmqpExchange string
        [exif] AMQP Exchange Name {FIBR_EXIF_AMQP_EXCHANGE} (default "fibr")
  -exifAmqpRoutingKey string
        [exif] AMQP Routing Key for exif {FIBR_EXIF_AMQP_ROUTING_KEY} (default "exif_input")
  -exifDirectAccess
        [exif] Use Exas with direct access to filesystem (no large file upload, send a GET request, Basic Auth recommended) {FIBR_EXIF_DIRECT_ACCESS}
  -exifMaxSize int
        [exif] Max file size (in bytes) for extracting exif (0 to no limit). Not used if DirectAccess enabled. {FIBR_EXIF_MAX_SIZE} (default 209715200)
  -exifPassword string
        [exif] Exif Tool URL Basic Password {FIBR_EXIF_PASSWORD}
  -exifURL string
        [exif] Exif Tool URL (exas) {FIBR_EXIF_URL} (default "http://exas:1080")
  -exifUser string
        [exif] Exif Tool URL Basic User {FIBR_EXIF_USER}
  -frameOptions string
        [owasp] X-Frame-Options {FIBR_FRAME_OPTIONS} (default "SAMEORIGIN")
  -graceDuration string
        [http] Grace duration when SIGTERM received {FIBR_GRACE_DURATION} (default "30s")
  -hsts
        [owasp] Indicate Strict Transport Security {FIBR_HSTS} (default true)
  -idleTimeout string
        [server] Idle Timeout {FIBR_IDLE_TIMEOUT} (default "2m")
  -ignorePattern string
        [crud] Ignore pattern when listing files or directory {FIBR_IGNORE_PATTERN}
  -key string
        [server] Key file {FIBR_KEY}
  -loggerJson
        [logger] Log format as JSON {FIBR_LOGGER_JSON}
  -loggerLevel string
        [logger] Logger level {FIBR_LOGGER_LEVEL} (default "INFO")
  -loggerLevelKey string
        [logger] Key for level in JSON {FIBR_LOGGER_LEVEL_KEY} (default "level")
  -loggerMessageKey string
        [logger] Key for message in JSON {FIBR_LOGGER_MESSAGE_KEY} (default "message")
  -loggerTimeKey string
        [logger] Key for timestamp in JSON {FIBR_LOGGER_TIME_KEY} (default "time")
  -minify
        Minify HTML {FIBR_MINIFY} (default true)
  -noAuth
        [auth] Disable basic authentification {FIBR_NO_AUTH}
  -okStatus int
        [http] Healthy HTTP Status code {FIBR_OK_STATUS} (default 204)
  -pathPrefix string
        Root Path Prefix {FIBR_PATH_PREFIX}
  -port uint
        [server] Listen port (0 to disable) {FIBR_PORT} (default 1080)
  -prometheusAddress string
        [prometheus] Listen address {FIBR_PROMETHEUS_ADDRESS}
  -prometheusCert string
        [prometheus] Certificate file {FIBR_PROMETHEUS_CERT}
  -prometheusGzip
        [prometheus] Enable gzip compression of metrics output {FIBR_PROMETHEUS_GZIP}
  -prometheusIdleTimeout string
        [prometheus] Idle Timeout {FIBR_PROMETHEUS_IDLE_TIMEOUT} (default "10s")
  -prometheusIgnore string
        [prometheus] Ignored path prefixes for metrics, comma separated {FIBR_PROMETHEUS_IGNORE}
  -prometheusKey string
        [prometheus] Key file {FIBR_PROMETHEUS_KEY}
  -prometheusPort uint
        [prometheus] Listen port (0 to disable) {FIBR_PROMETHEUS_PORT} (default 9090)
  -prometheusReadTimeout string
        [prometheus] Read Timeout {FIBR_PROMETHEUS_READ_TIMEOUT} (default "5s")
  -prometheusShutdownTimeout string
        [prometheus] Shutdown Timeout {FIBR_PROMETHEUS_SHUTDOWN_TIMEOUT} (default "5s")
  -prometheusWriteTimeout string
        [prometheus] Write Timeout {FIBR_PROMETHEUS_WRITE_TIMEOUT} (default "10s")
  -publicURL string
        Public URL {FIBR_PUBLIC_URL} (default "http://localhost:1080")
  -readTimeout string
        [server] Read Timeout {FIBR_READ_TIMEOUT} (default "2m")
  -sanitizeOnStart
        [crud] Sanitize name on start {FIBR_SANITIZE_ON_START}
  -shareAmqpExchange string
        [share] AMQP Exchange Name {FIBR_SHARE_AMQP_EXCHANGE} (default "fibr.shares")
  -shareAmqpExclusiveRoutingKey string
        [share] AMQP Routing Key for exclusive lock on default exchange {FIBR_SHARE_AMQP_EXCLUSIVE_ROUTING_KEY} (default "fibr.semaphore.shares")
  -shareAmqpRoutingKey string
        [share] AMQP Routing Key for share {FIBR_SHARE_AMQP_ROUTING_KEY} (default "share")
  -shutdownTimeout string
        [server] Shutdown Timeout {FIBR_SHUTDOWN_TIMEOUT} (default "10s")
  -storageAccessKey string
        [storage] Storage Object Access Key {FIBR_STORAGE_ACCESS_KEY}
  -storageBucket string
        [storage] Storage Object Bucket {FIBR_STORAGE_BUCKET}
  -storageDirectory string
        [storage] Path to directory {FIBR_STORAGE_DIRECTORY} (default "/data")
  -storageEndpoint string
        [storage] Storage Object endpoint {FIBR_STORAGE_ENDPOINT}
  -storageSSL
        [storage] Use SSL {FIBR_STORAGE_SSL} (default true)
  -storageSecretAccess string
        [storage] Storage Object Secret Access {FIBR_STORAGE_SECRET_ACCESS}
  -thumbnailAmqpExchange string
        [thumbnail] AMQP Exchange Name {FIBR_THUMBNAIL_AMQP_EXCHANGE} (default "fibr")
  -thumbnailAmqpStreamRoutingKey string
        [thumbnail] AMQP Routing Key for stream {FIBR_THUMBNAIL_AMQP_STREAM_ROUTING_KEY} (default "stream")
  -thumbnailAmqpThumbnailRoutingKey string
        [thumbnail] AMQP Routing Key for thumbnail {FIBR_THUMBNAIL_AMQP_THUMBNAIL_ROUTING_KEY} (default "thumbnail")
  -thumbnailDirectAccess
        [thumbnail] Use Vith with direct access to filesystem (no large file upload, send a GET request, Basic Auth recommended) {FIBR_THUMBNAIL_DIRECT_ACCESS}
  -thumbnailLargeSize uint
        [thumbnail] Size of large thumbnail for story display (thumbnail are always squared). 0 to disable {FIBR_THUMBNAIL_LARGE_SIZE} (default 800)
  -thumbnailMaxSize int
        [thumbnail] Maximum file size (in bytes) for generating thumbnail (0 to no limit). Not used if DirectAccess enabled. {FIBR_THUMBNAIL_MAX_SIZE} (default 209715200)
  -thumbnailMinBitrate uint
        [thumbnail] Minimal video bitrate (in bits per second) to generate a streamable version (in HLS), if DirectAccess enabled {FIBR_THUMBNAIL_MIN_BITRATE} (default 80000000)
  -thumbnailPassword string
        [thumbnail] Vith Thumbnail Basic Auth Password {FIBR_THUMBNAIL_PASSWORD}
  -thumbnailURL string
        [thumbnail] Vith Thumbnail URL {FIBR_THUMBNAIL_URL} (default "http://vith:1080")
  -thumbnailUser string
        [thumbnail] Vith Thumbnail Basic Auth User {FIBR_THUMBNAIL_USER}
  -title string
        Application title {FIBR_TITLE} (default "fibr")
  -tracerRate string
        [tracer] Jaeger sample rate, 'always', 'never' or a float value {FIBR_TRACER_RATE} (default "always")
  -tracerURL string
        [tracer] Jaeger endpoint URL (e.g. http://jaeger:14268/api/traces) {FIBR_TRACER_URL}
  -url string
        [alcotest] URL to check {FIBR_URL}
  -userAgent string
        [alcotest] User-Agent for check {FIBR_USER_AGENT} (default "Alcotest")
  -webhookAmqpExchange string
        [webhook] AMQP Exchange Name {FIBR_WEBHOOK_AMQP_EXCHANGE} (default "fibr.webhooks")
  -webhookAmqpExclusiveRoutingKey string
        [webhook] AMQP Routing Key for exclusive lock on default exchange {FIBR_WEBHOOK_AMQP_EXCLUSIVE_ROUTING_KEY} (default "fibr.semaphore.webhooks")
  -webhookAmqpRoutingKey string
        [webhook] AMQP Routing Key for webhook {FIBR_WEBHOOK_AMQP_ROUTING_KEY} (default "webhook")
  -webhookSecret string
        [webhook] Secret for HMAC Signature {FIBR_WEBHOOK_SECRET}
  -writeTimeout string
        [server] Write Timeout {FIBR_WRITE_TIMEOUT} (default "2m")
```

# Caveats

## Multiples instances

Fibr doesn't handle multiple instances running at the same time on the same `rootFolder`, if you use [Sharing feature](#sharing).

Shares' metadatas are stored in a file, loaded at the start of the application. If an _instance A_ adds a share, _instance B_ can't see it. If they are both behind the same load-balancer, it can leads to an erratic behavior. Fibr has also an internal cron that purge expired shares and write the new metadatas to the file. If _instance A_ adds a share and _instance B_ runs the cron, the share added in _instance A_ is lost.

If you enable AMQP, it can handle thoses behaviours by using an exclusive lock with an AMQP semaphore mechanism.
