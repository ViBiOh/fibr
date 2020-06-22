# fibr

Web File Browser and Manager.

* Lightweight (11MB self-sufficient binary, low memory consumption at runtime).
* Mobile-first interface.
* Thumbnail generation for Image, PDF and Video (with help of sidecar)
* Works in pure HTML or with very little javascript for improved file upload
* Can share directory with ou without password and with or without edit right.
* Support multiple storage backend (basic filesystem implemented, but can be anything like Minio or S3)

[![Build Status](https://travis-ci.com/ViBiOh/fibr.svg?branch=master)](https://travis-ci.com/ViBiOh/fibr)
[![Go Report Card](https://goreportcard.com/badge/github.com/ViBiOh/fibr)](https://goreportcard.com/report/github.com/ViBiOh/fibr)
[![codecov](https://codecov.io/gh/ViBiOh/fibr/branch/master/graph/badge.svg)](https://codecov.io/gh/ViBiOh/fibr)
[![Quality Gate Status](https://sonarcloud.io/api/project_badges/measure?project=ViBiOh_fibr&metric=alert_status)](https://sonarcloud.io/dashboard?id=ViBiOh_fibr)

Thanks to [FontAwesome](https://fontawesome.com) for providing awesome svg.

## Getting started

### Users

Users can be declared in-memory with a bcrypted format

```bash
htpasswd -nBb admin password
```

Users must have `admin` profile in order to be accepted.

### As a binary, without authentification

```bash
go get github.com/ViBiOh/fibr/cmd/fibr
fibr \
  -noAuth \
  -fsDirectory "$(pwd)" \
  -publicURL "http://localhost:1080" \
  -csp "default-src 'self'; base-uri 'self'; script-src 'unsafe-inline'; style-src 'unsafe-inline'; img-src 'self' data:"
```

### As a single Docker container, with admin/password user

```bash
docker pull vibioh/fibr
docker run -d \
  -p 1080:180/tcp \
  --name fibr \
  -v ${PWD}:/data/ \
  -e FIBR_PUBLIC_URL="http://localhost:1080" \
  -e FIBR_CSP="default-src 'self'; base-uri 'self'; script-src 'unsafe-inline'; style-src 'unsafe-inline'; img-src 'self' data:" \
  -e FIBR_AUTH_PROFILES="1:admin" \
  -e FIBR_AUTH_USERS="1:$(htpasswd -nBb admin password)" \
  fibr
```

### As a docker-compose stack

You can inspire yourself from the [docker-compose.yml](docker-compose.yml) file I personnaly use.

## CLI Usage

```bash
Usage of fibr:
  -address string
        [http] Listen address {FIBR_ADDRESS}
  -authProfiles string
        [auth] Users profiles in the form 'id:profile1|profile2,id2:profile1' {FIBR_AUTH_PROFILES}
  -authUsers string
        [auth] Users credentials in the form 'id:login:password,id2:login2:password2' {FIBR_AUTH_USERS}
  -cert string
        [http] Certificate file {FIBR_CERT}
  -csp string
        [owasp] Content-Security-Policy {FIBR_CSP} (default "default-src 'self'; base-uri 'self'")
  -frameOptions string
        [owasp] X-Frame-Options {FIBR_FRAME_OPTIONS} (default "deny")
  -fsDirectory string
        [fs] Path to served directory {FIBR_FS_DIRECTORY} (default "/data")
  -graceDuration string
        [http] Grace duration when SIGTERM received {FIBR_GRACE_DURATION} (default "15s")
  -hsts
        [owasp] Indicate Strict Transport Security {FIBR_HSTS} (default true)
  -ignorePattern string
        [crud] Ignore pattern when listing files or directory {FIBR_IGNORE_PATTERN}
  -key string
        [http] Key file {FIBR_KEY}
  -metadata
        [crud] Enable metadata storage {FIBR_METADATA} (default true)
  -noAuth
        [auth] Disable basic authentification {FIBR_NO_AUTH}
  -okStatus int
        [http] Healthy HTTP Status code {FIBR_OK_STATUS} (default 204)
  -port uint
        [http] Listen port {FIBR_PORT} (default 1080)
  -prometheusPath string
        [prometheus] Path for exposing metrics {FIBR_PROMETHEUS_PATH} (default "/metrics")
  -publicURL string
        [fibr] Public URL {FIBR_PUBLIC_URL} (default "https://fibr.vibioh.fr")
  -sanitizeOnStart
        [crud] Sanitize name on start {FIBR_SANITIZE_ON_START}
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
```
