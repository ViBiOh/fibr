# fibr

Web File Browser and Manager.

- Lightweight (11MB self-sufficient binary, low memory consumption at runtime).
- Mobile-first interface, with light payload. Dark themed.
- Thumbnail generation for image, PDF and video (with help of sidecars)
- Works in pure HTML or with very little javascript for improved file upload
- Can share directory with ou without password and with or without edit right.
- Support for basic filesystem and storage object (in beta)

![](docs/fibr.png)

[![Build](https://github.com/ViBiOh/fibr/workflows/Build/badge.svg)](https://github.com/ViBiOh/fibr/actions)
[![codecov](https://codecov.io/gh/ViBiOh/fibr/branch/main/graph/badge.svg)](https://codecov.io/gh/ViBiOh/fibr)
[![Quality Gate Status](https://sonarcloud.io/api/project_badges/measure?project=ViBiOh_fibr&metric=alert_status)](https://sonarcloud.io/dashboard?id=ViBiOh_fibr)

Thanks to [FontAwesome](https://fontawesome.com) for providing awesome svg.

## Concepts

Fibr aims to provide simple browsing of your filesystem. It's a single static binary with embedded html templates. No Javascript framework. HTTP and HTML have all we need.

Fibr aims to be compatible with the most platforms available, on a best-effort basis. Fibr itself is already compatible with `x86_64`, `arm`, `arm64` architectures. But sidecars, which depends on system library, are not all ready yet.

### Folder

Fibr browses files of given `-data` option folder (or S3 configuration), called "root folder". For security reason, it's not possible to browse parent.

It aims to be consistent accross all existing filesystem (block storage, object storage, etc.) and thus enforces filenames in lowercase, with no space or special character. At start, it walks every files and reports names that breaks its policy. It doesn't modify existing files unless you set `-sanitizeOnStart` option.

Fibr creates a `.fibr` folder in _root folder_ for storing its metadata: shares' configuration, thumbnails and exif. If you want to stop using _fibr_ or start with a fresh installation (e.g. regenerating thumbnails), you can delete this folder.

### Sidecars

Fibr generates thumbnails of images, PDF and videos when these [mime-types are detected](https://developer.mozilla.org/en-US/docs/Web/HTTP/Basics_of_HTTP/MIME_types/Common_types) and sidecars are provided. Sidecars are [h2non/imaginary](https://github.com/h2non/imaginary), [ViBiOh/vith](https://github.com/vibioh/vith) and [ViBiOh/exas](https://github.com/vibioh/exas).

You can refer to these projects for installing and configuring them and set `-thumbnailImageURL`, `-thumbnailVideoURL` and `-exifURL` options.

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

It will contains an extra key `new` with the same structure of `item` in case of a `rename` event.

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

With help of different sidecars, Fibr can generate image, video and PDF thumbnails. These sidecars can be self hosted with ease. It can also extract and enrich content displayed by looking at [EXIF Data](https://en.wikipedia.org/wiki/Exif), also with the help of a little sidecar. This behaviour are opt-out (if you remove the `url` of the service, Fibr will do nothing).

For the last mile, Fibr can try to reverse geocoding the GPS data found in EXIF, using [Open Street Map](https://wiki.openstreetmap.org/wiki/Nominatim). Self-hosting this kind of service can be complicated and calling a third-party party with such sensible datas is an opt-in decision.

## Getting started

### As a binary, without authentification

This is for local purpose with simple and hassle-free sharing in a private network.

```bash
go install github.com/ViBiOh/fibr/cmd/fibr@latest
fibr \
  -noAuth \
  -fsDirectory "$(pwd)" \
  -thumbnailImageURL "" \
  -thumbnailVideoURL "" \
  -exifURL "" \
  -publicURL "http://localhost:1080"
```

### As a single Docker container, with admin/password user

For long-living sharing with password and self-contained app in Docker, with no thumbnail generation or exif, configured with environment variables.

```bash
docker run -d \
  -p 1080:180/tcp \
  --name fibr \
  -v ${PWD}:/data/ \
  -e FIBR_PUBLIC_URL="http://localhost:1080" \
  -e FIBR_AUTH_PROFILES="1:admin" \
  -e FIBR_AUTH_USERS="1:$(htpasswd -nBb login password)" \
  -e FIBR_THUMBNAIL_IMAGE_URL="" \
  -e FIBR_THUMBNAIL_VIDEO_URL="" \
  -e FIBR_EXIF_URL="" \
  vibioh/fibr
```

### As a docker-compose / k8s stack

For prod-ready run with thumbnails generation of image, PDF and videos, _this is the recommended approach_.

You can inspire yourself from the [docker-compose.example.yaml](docker-compose.example.yaml) file I personnaly used. Beware of `-authUsers` option: bcrypted passwords contain dollar sign, which `docker-compose` tries to resolve as a shell variable, [you must escape it](https://docs.docker.com/compose/compose-file/compose-file-v2/#variable-substitution).

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
  -authProfiles string
        [auth] Users profiles in the form 'id:profile1|profile2,id2:profile1' {FIBR_AUTH_PROFILES}
  -authUsers string
        [auth] Users credentials in the form 'id:login:password,id2:login2:password2' {FIBR_AUTH_USERS}
  -cert string
        [server] Certificate file {FIBR_CERT}
  -csp string
        [owasp] Content-Security-Policy {FIBR_CSP} (default "default-src 'self'; base-uri 'self'; script-src 'unsafe-inline'; style-src 'unsafe-inline'; img-src 'self' data:")
  -exifAggregateOnStart
        [exif] Aggregate EXIF data per folder on start {FIBR_EXIF_AGGREGATE_ON_START}
  -exifDateOnStart
        [exif] Change file date from EXIF date on start {FIBR_EXIF_DATE_ON_START}
  -exifDirectAccess
        [exif] Use Exas with direct access to filesystem (no large file upload to it, send a GET request) {FIBR_EXIF_DIRECT_ACCESS}
  -exifGeocodeURL string
        [exif] Nominatim Geocode Service URL. This can leak GPS metadatas to a third-party (e.g. "https://nominatim.openstreetmap.org") {FIBR_EXIF_GEOCODE_URL}
  -exifMaxSize int
        [exif] Max file size (in bytes) for extracting exif (0 to no limit) {FIBR_EXIF_MAX_SIZE} (default 209715200)
  -exifPassword string
        [exif] Exif Tool URL Basic Password {FIBR_EXIF_PASSWORD}
  -exifURL string
        [exif] Exif Tool URL (exas) {FIBR_EXIF_URL} (default "http://exas:1080")
  -exifUser string
        [exif] Exif Tool URL Basic User {FIBR_EXIF_USER}
  -frameOptions string
        [owasp] X-Frame-Options {FIBR_FRAME_OPTIONS} (default "SAMEORIGIN")
  -fsDirectory string
        [fs] Path to served directory {FIBR_FS_DIRECTORY} (default "/data")
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
        Public URL {FIBR_PUBLIC_URL} (default "https://fibr.vibioh.fr")
  -readTimeout string
        [server] Read Timeout {FIBR_READ_TIMEOUT} (default "2m")
  -s3AccessKey string
        [s3] Storage Object Access Key {FIBR_S3_ACCESS_KEY}
  -s3Bucket string
        [s3] Storage Object Bucket {FIBR_S3_BUCKET}
  -s3Endpoint string
        [s3] Storage Object endpoint {FIBR_S3_ENDPOINT}
  -s3SSL
        [s3] Use SSL {FIBR_S3_SSL} (default true)
  -s3SecretAccess string
        [s3] Storage Object Secret Access {FIBR_S3_SECRET_ACCESS}
  -sanitizeOnStart
        [crud] Sanitize name on start {FIBR_SANITIZE_ON_START}
  -share
        [share] Enable sharing feature {FIBR_SHARE} (default true)
  -shutdownTimeout string
        [server] Shutdown Timeout {FIBR_SHUTDOWN_TIMEOUT} (default "10s")
  -thumbnailDirectAccess
        [thumbnail] Use Vith with direct access to filesystem (no large file upload to it, send a GET request, Basic Auth recommended) {FIBR_THUMBNAIL_DIRECT_ACCESS}
  -thumbnailImagePassword string
        [thumbnail] Imaginary Basic Auth Password {FIBR_THUMBNAIL_IMAGE_PASSWORD}
  -thumbnailImageURL string
        [thumbnail] Imaginary URL {FIBR_THUMBNAIL_IMAGE_URL} (default "http://image:9000")
  -thumbnailImageUser string
        [thumbnail] Imaginary Basic Auth User {FIBR_THUMBNAIL_IMAGE_USER}
  -thumbnailMaxSize int
        [thumbnail] Maximum file size (in bytes) for generating thumbnail (0 to no limit) {FIBR_THUMBNAIL_MAX_SIZE} (default 209715200)
  -thumbnailMinBitrate uint
        [thumbnail] Minimal video bitrate (in bits per second) to generate a streamable version (in HLS), if DirectAccess enabled {FIBR_THUMBNAIL_MIN_BITRATE} (default 80000000)
  -thumbnailVideoPassword string
        [thumbnail] Video Thumbnail Basic Auth Password {FIBR_THUMBNAIL_VIDEO_PASSWORD}
  -thumbnailVideoURL string
        [thumbnail] Video Thumbnail URL {FIBR_THUMBNAIL_VIDEO_URL} (default "http://video:1080")
  -thumbnailVideoUser string
        [thumbnail] Video Thumbnail Basic Auth User {FIBR_THUMBNAIL_VIDEO_USER}
  -title string
        Application title {FIBR_TITLE} (default "fibr")
  -url string
        [alcotest] URL to check {FIBR_URL}
  -userAgent string
        [alcotest] User-Agent for check {FIBR_USER_AGENT} (default "Alcotest")
  -webhookEnabled
        [webhook] Enable webhook feature {FIBR_WEBHOOK_ENABLED} (default true)
  -webhookSecret string
        [webhook] Secret for HMAC Signature {FIBR_WEBHOOK_SECRET}
  -writeTimeout string
        [server] Write Timeout {FIBR_WRITE_TIMEOUT} (default "2m")
```

# Caveats

## Multiples instances

Fibr doesn't handle multiple instances running at the same time on the same `rootFolder`, if you use [Sharing feature](#sharing).

Shares' metadatas are stored in a file, loaded at the start of the application. If an _instance A_ adds a share, _instance B_ can't see it. If they are both behind the same load-balancer, it can leads to an erratic behavior.

Fibr has also an internal cron that purge expired shares and write the new metadatas to the file. If _instance A_ adds a share and _instance B_ runs the cron, the share added in _instance A_ is lost. It's a known limitation I need to work on, without adding an external tool like Redis and without being I/O intensive on filesystem.
