# fibr

Web File Browser and Manager

[![Build Status](https://travis-ci.org/ViBiOh/fibr.svg?branch=master)](https://travis-ci.org/ViBiOh/fibr)
[![Go Report Card](https://goreportcard.com/badge/github.com/ViBiOh/fibr)](https://goreportcard.com/report/github.com/ViBiOh/fibr)

Thanks to [FontAwesome](https://fontawesome.com) for providing awesome svg.

## Installation

```bash
go get github.com/ViBiOh/fibr/cmd
```

## Usage

```bash
Usage of fibr:
  -authDisable
        [auth] Disable auth
  -authUrl string
        [auth] Auth URL, if remote
  -authUsers string
        [auth] Allowed users and profiles (e.g. user:profile1|profile2,user2:profile3). Empty allow any identified user
  -basicUsers id:username:password,id2:username2:password2
        [basic] Users in the form id:username:password,id2:username2:password2
  -cert string
        [http] Certificate file
  -csp string
        [owasp] Content-Security-Policy (default "default-src 'self'; base-uri 'self'")
  -frameOptions string
        [owasp] X-Frame-Options (default "deny")
  -fsDirectory string
        [filesystem] Path to served directory (default "/data")
  -hsts
        [owasp] Indicate Strict Transport Security (default true)
  -key string
        [http] Key file
  -metadata
        Enable metadata storage (default true)
  -port int
        [http] Listen port (default 1080)
  -prometheusPath string
        [prometheus] Path for exposing metrics (default "/metrics")
  -publicURL string
        [fibr] Public URL (default "https://fibr.vibioh.fr")
  -thumbnailImaginaryURL string
        [thumbnail] Imaginary URL (default "http://image:9000")
  -tracingAgent string
        [tracing] Jaeger Agent (e.g. host:port) (default "jaeger:6831")
  -tracingName string
        [tracing] Service name
  -url string
        [alcotest] URL to check
  -userAgent string
        [alcotest] User-Agent for check (default "Golang alcotest")
  -version string
        [fibr] Version (used mainly as a cache-buster)
```
