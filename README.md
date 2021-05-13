# fibr

Web File Browser and Manager.

- Lightweight (11MB self-sufficient binary, low memory consumption at runtime).
- Mobile-first interface, with light payload. Dark themed.
- Thumbnail generation for image, PDF and video (with help of sidecars)
- Works in pure HTML or with very little javascript for improved file upload
- Can share directory with ou without password and with or without edit right.
- Support multiple storage backend (basic filesystem implemented, but can be anything like Minio or S3)

![](docs/fibr.png)

[![Build](https://github.com/ViBiOh/fibr/workflows/Build/badge.svg)](https://github.com/ViBiOh/fibr/actions)
[![codecov](https://codecov.io/gh/ViBiOh/fibr/branch/main/graph/badge.svg)](https://codecov.io/gh/ViBiOh/fibr)
[![Quality Gate Status](https://sonarcloud.io/api/project_badges/measure?project=ViBiOh_fibr&metric=alert_status)](https://sonarcloud.io/dashboard?id=ViBiOh_fibr)

Thanks to [FontAwesome](https://fontawesome.com) for providing awesome svg.

## Concepts

Fibr aims to provide simple browsing of your filesystem. It's a single static binary with embed html templates. No Javascript framework. HTTP and HTML have all we need.

Fibr aims to be compatible with the most platforms available, on a best-effort basis. Fibr itself is already compatible with `x86_64`, `arm`, `arm64` architectures. But sidecars, which depends on system library, are not all ready yet.

### Folder

Fibr browses files of given `-data` option folder, called "root folder". For security reason, it's not possible to browse parent.

It aims to be consistent accross all existing filesystem (block storage, object storage, etc.) and thus enforces filenames in lowercase, with no space or special character. At start, it walks every files and reports names that breaks its policy. It doesn't modify existing files unless you set `-sanitizeOnStart` option.

Fibr creates a `.fibr` folder in _root folder_ for storing its metadata: shares' configuration and thumbnails. If you want to stop using _fibr_ or start with a fresh installation (e.g. regenerating thumbnails), you can delete this folder.

### Files

Fibr generates thumbnails of images, PDF and videos when these [mime-types are detected](https://developer.mozilla.org/en-US/docs/Web/HTTP/Basics_of_HTTP/MIME_types/Common_types) and sidecars are provided. Sidecars are [h2non/imaginary](https://github.com/h2non/imaginary) and [ViBiOh/vith](https://github.com/vibioh/vith).

You can refer to these projects for installing and configuring them and set `-thumbnailImageURL` and `-thumbnailVideoURL` options.

### Security

Authentication is made with [Basic Auth](https://developer.mozilla.org/en-US/docs/Web/HTTP/Authentication), compatible with all browsers and CLI tools such as `curl`. I _strongly recommend configuring HTTPS_ in order to avoid exposing your credentials in plain text.

You can set `-cert` and `-key` options, it uses [Golang's standard net/http#ServeTLS](https://golang.org/pkg/net/http/#ServeTLS) under the hood.

You can also configure a reverse proxy with Let's Encrypt to manage encryption, such as [Traefik](https://docs.traefik.io).

### Sharing

You can share folders or just one file: it generates a short link that gain access to shared object and is considered as "root folder" with no parent escalation.

It can be password-protected: user _has to_ enter password to see content (login is not used, you can leave it blank).

It can be read-only or with edit right. With edit-right, user can do anything as you, uploading, deleting, renaming, except generating new shares.

> It's really useful for sharing files with friends. You don't need account at Google, Dropbox, iCloud or a mobile-app: a link and everyone can see and share content!

This is the main reason I've started to develop this app.

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

In order to work, your user _must have_ `admin` profile sets with the `-authProfiles` option.

## Getting started

### As a binary, without authentification

This is for local purpose with simple and hassle-free sharing in a private network.

```bash
go get github.com/ViBiOh/fibr/cmd/fibr
fibr \
  -noAuth \
  -fsDirectory "$(pwd)" \
  -publicURL "http://localhost:1080"
```

### As a single Docker container, with admin/password user

For long-living sharing with password and self-contained app in Docker, with no thumbnail generation.

```bash
docker run -d \
  -p 1080:180/tcp \
  --name fibr \
  -v ${PWD}:/data/ \
  -e FIBR_PUBLIC_URL="http://localhost:1080" \
  -e FIBR_AUTH_PROFILES="1:admin" \
  -e FIBR_AUTH_USERS="1:$(htpasswd -nBb login password)" \
  vibioh/fibr
```

### As a docker-compose / k8s stack

For prod-ready run with thumbnails generation of image, PDF and videos, _this is the recommended approach_.

You can inspire yourself from the [docker-compose.example.yaml](docker-compose.example.yaml) file I personnaly used. Beware of `-authUsers` option: bcrypted passwords contain dollar sign, which `docker-compose` tries to resolve as a shell variable, [you must escape it](https://docs.docker.com/compose/compose-file/compose-file-v2/#variable-substitution).

You'll find a Kubernetes exemple in the [`infra/`](infra/) folder, using my [`app chart`](https://github.com/ViBiOh/charts/tree/main/app). My personnal k8s runs on `arm64` and thumbnail converters are not yet ready for this architecture, so I use a mix of `helm` and `docker-compose.yaml`.

## Endpoints

- `GET /health`: healthcheck of server, respond [`okStatus (default 204)`](#usage) or `503` during [`graceDuration`](#usage) when SIGTERM is received
- `GET /ready`: same response than `/health` but it also checks external dependencies availability
- `GET /version`: value of `VERSION` environment variable
- `GET /metrics`: Prometheus metrics values, on a dedicated port

## Usage

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
  -metadata
        [metadata] Enable metadata storage {FIBR_METADATA} (default true)
  -noAuth
        [auth] Disable basic authentification {FIBR_NO_AUTH}
  -okStatus int
        [http] Healthy HTTP Status code {FIBR_OK_STATUS} (default 204)
  -port uint
        [server] Listen port {FIBR_PORT} (default 1080)
  -prometheusAddress string
        [prometheus] Listen address {FIBR_PROMETHEUS_ADDRESS}
  -prometheusCert string
        [prometheus] Certificate file {FIBR_PROMETHEUS_CERT}
  -prometheusIdleTimeout string
        [prometheus] Idle Timeout {FIBR_PROMETHEUS_IDLE_TIMEOUT} (default "10s")
  -prometheusIgnore string
        [prometheus] Ignored path prefixes for metrics, comma separated {FIBR_PROMETHEUS_IGNORE}
  -prometheusKey string
        [prometheus] Key file {FIBR_PROMETHEUS_KEY}
  -prometheusPort uint
        [prometheus] Listen port {FIBR_PROMETHEUS_PORT} (default 9090)
  -prometheusReadTimeout string
        [prometheus] Read Timeout {FIBR_PROMETHEUS_READ_TIMEOUT} (default "5s")
  -prometheusShutdownTimeout string
        [prometheus] Shutdown Timeout {FIBR_PROMETHEUS_SHUTDOWN_TIMEOUT} (default "5s")
  -prometheusWriteTimeout string
        [prometheus] Write Timeout {FIBR_PROMETHEUS_WRITE_TIMEOUT} (default "10s")
  -publicURL string
        [fibr] Public URL {FIBR_PUBLIC_URL} (default "https://fibr.vibioh.fr")
  -readTimeout string
        [server] Read Timeout {FIBR_READ_TIMEOUT} (default "2m")
  -sanitizeOnStart
        [crud] Sanitize name on start {FIBR_SANITIZE_ON_START}
  -shutdownTimeout string
        [server] Shutdown Timeout {FIBR_SHUTDOWN_TIMEOUT} (default "10s")
  -thumbnailImageURL string
        [thumbnail] Imaginary URL {FIBR_THUMBNAIL_IMAGE_URL} (default "http://image:9000")
  -thumbnailVideoURL string
        [thumbnail] Video Thumbnail URL {FIBR_THUMBNAIL_VIDEO_URL} (default "http://video:1080")
  -url string
        [alcotest] URL to check {FIBR_URL}
  -userAgent string
        [alcotest] User-Agent for check {FIBR_USER_AGENT} (default "Alcotest")
  -version string
        [fibr] Version (used mainly as a cache-buster) {FIBR_VERSION}
  -writeTimeout string
        [server] Write Timeout {FIBR_WRITE_TIMEOUT} (default "2m")
```

# Caveats

## Multiples instances

Fibr doesn't handle multiple instances running at the same time on the same `rootFolder`, if you use [Sharing feature](#sharing).

Shares' metadatas are stored in a file, loaded at the start of the application. If an _instance A_ adds a share, _instance B_ can't see it. If they are both behind the same load-balancer, it can leads to an erratic behavior.

Fibr has also an internal cron that purge expired shares and write the new metadatas to the file. If _instance A_ adds a share and _instance B_ runs the cron, the share added in _instance A_ is lost. It's a known limitation I need to work on, without adding an external tool like Redis and without being I/O intensive on filesystem.

## Change in metadatas format

In recent version, metadatas file structure has changed from an array to the map. You can convert it with the following snippet (make a copy before, updating a JSON in place with `jq` can be risky).

```bash
cat <<<"$(jq 'map({ (.id): . }) | add' .json)" > .json
```
