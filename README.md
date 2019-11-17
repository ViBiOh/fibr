# fibr

Web File Browser and Manager

[![Build Status](https://travis-ci.org/ViBiOh/fibr.svg?branch=master)](https://travis-ci.org/ViBiOh/fibr)
[![Go Report Card](https://goreportcard.com/badge/github.com/ViBiOh/fibr)](https://goreportcard.com/report/github.com/ViBiOh/fibr)
[![codecov](https://codecov.io/gh/ViBiOh/fibr/branch/master/graph/badge.svg)](https://codecov.io/gh/ViBiOh/fibr)
[![Dependabot Status](https://api.dependabot.com/badges/status?host=github&repo=ViBiOh/fibr)](https://dependabot.com)

Thanks to [FontAwesome](https://fontawesome.com) for providing awesome svg.

## Installation

```bash
go get github.com/ViBiOh/fibr/cmd
```

## Usage

```bash
Usage of fibr:
  -address string
        [http] Listen address {FIBR_ADDRESS}
  -authDisable
        [auth] Disable auth {FIBR_AUTH_DISABLE}
  -authUrl string
        [auth] Auth URL, if remote {FIBR_AUTH_URL}
  -authUsers string
        [auth] Allowed users and profiles (e.g. user:profile1|profile2,user2:profile3). Empty allow any identified user {FIBR_AUTH_USERS}
  -basicUsers id:username:password,id2:username2:password2
        [basic] Users in the form id:username:password,id2:username2:password2 {FIBR_BASIC_USERS}
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
  -port int
        [http] Listen port {FIBR_PORT} (default 1080)
  -prometheusPath string
        [prometheus] Path for exposing metrics {FIBR_PROMETHEUS_PATH} (default "/metrics")
  -publicURL string
        [fibr] Public URL {FIBR_PUBLIC_URL} (default "https://fibr.vibioh.fr")
  -thumbnailImaginaryURL string
        [thumbnail] Imaginary URL {FIBR_THUMBNAIL_IMAGINARY_URL} (default "http://image:9000")
  -url string
        [alcotest] URL to check {FIBR_URL}
  -userAgent string
        [alcotest] User-Agent for check {FIBR_USER_AGENT} (default "Alcotest")
  -version string
        [fibr] Version (used mainly as a cache-buster) {FIBR_VERSION}
```
