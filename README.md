# fibr

Web File Browser and Manager.

* Lightweight (11MB self-sufficient binary, low memory consumption at runtime).
* Mobile-first interface.
* Thumbnail generation for Image, PDF and Video (with help of sidecar)
* Works in pure HTML or with very little javascript for improved file upload
* Can share directory with ou without password and with or without edit right.
* Support multiple storage backend (basic filesystem implemented, but can be anything like Minio or S3)

[![Build Status](https://travis-ci.org/ViBiOh/fibr.svg?branch=master)](https://travis-ci.org/ViBiOh/fibr)
[![Go Report Card](https://goreportcard.com/badge/github.com/ViBiOh/fibr)](https://goreportcard.com/report/github.com/ViBiOh/fibr)
[![codecov](https://codecov.io/gh/ViBiOh/fibr/branch/master/graph/badge.svg)](https://codecov.io/gh/ViBiOh/fibr)
[![Dependabot Status](https://api.dependabot.com/badges/status?host=github&repo=ViBiOh/fibr)](https://dependabot.com)

Thanks to [FontAwesome](https://fontawesome.com) for providing awesome svg.

## Installation

```bash
go get github.com/ViBiOh/fibr/cmd/fibr
```

## Usage

### File Browser (fibr)

```bash
Usage of fibr:
  -address string
        [http] Listen address {FIBR_ADDRESS}
  -authProfiles id:profile1|profile2,id2:profile1
        [auth] Users profiles in the form id:profile1|profile2,id2:profile1 {FIBR_AUTH_PROFILES}
  -authUsers id:login:password,id2:login2:password2
        [auth] Users credentials in the form id:login:password,id2:login2:password2 {FIBR_AUTH_USERS}
  -cert string
        [http] Certificate file {FIBR_CERT}
  -csp string
        [owasp] Content-Security-Policy {FIBR_CSP} (default "default-src 'self'; base-uri 'self'")
  -frameOptions string
        [owasp] X-Frame-Options {FIBR_FRAME_OPTIONS} (default "deny")
  -fsDirectory string
        [fs] Path to served directory {FIBR_FS_DIRECTORY} (default "/data")
  -hsts
        [owasp] Indicate Strict Transport Security {FIBR_HSTS} (default true)
  -key string
        [http] Key file {FIBR_KEY}
  -metadata
        [crud] Enable metadata storage {FIBR_METADATA} (default true)
  -okStatus int
        [http] Healthy HTTP Status code {FIBR_OK_STATUS} (default 204)
  -port uint
        [http] Listen port {FIBR_PORT} (default 1080)
  -prometheusPath string
        [prometheus] Path for exposing metrics {FIBR_PROMETHEUS_PATH} (default "/metrics")
  -publicURL string
        [fibr] Public URL {FIBR_PUBLIC_URL} (default "https://fibr.vibioh.fr")
  -thumbnailImaginaryURL string
        [thumbnail] Imaginary URL {FIBR_THUMBNAIL_IMAGINARY_URL} (default "http://image:9000")
  -thumbnailVideoURL string
        [thumbnail] Video Thumbnail URL {FIBR_THUMBNAIL_VIDEO_URL} (default "http://video:1080")
  -url string
        [alcotest] URL to check {FIBR_URL}
  -userAgent string
        [alcotest] User-Agent for check {FIBR_USER_AGENT} (default "Alcotest")
  -version string
        [fibr] Version (used mainly as a cache-buster) {FIBR_VERSION}
```

### Video Thumbnail (vith)

```bash
Usage of vith:
  -address string
        [http] Listen address {VITH_ADDRESS}
  -cert string
        [http] Certificate file {VITH_CERT}
  -key string
        [http] Key file {VITH_KEY}
  -okStatus int
        [http] Healthy HTTP Status code {VITH_OK_STATUS} (default 204)
  -port uint
        [http] Listen port {VITH_PORT} (default 1080)
  -url string
        [alcotest] URL to check {VITH_URL}
  -userAgent string
        [alcotest] User-Agent for check {VITH_USER_AGENT} (default "Alcotest")
```
