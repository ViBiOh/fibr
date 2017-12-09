# fibr

Web File Browser and Manager

[![Build Status](https://travis-ci.org/ViBiOh/fibr.svg?branch=master)](https://travis-ci.org/ViBiOh/fibr)
[![Go Report Card](https://goreportcard.com/badge/github.com/ViBiOh/fibr)](https://goreportcard.com/report/github.com/ViBiOh/fibr)

Thanks to [FontAwesome](https://fontawesome.com) for providing awesome svg.

## Installation

```bash
go get -u github.com/ViBiOh/fibr
```

## Usage

```bash
  -authUrl string
      [auth] Auth URL
  -authUsers string
      [auth] List of allowed users and profiles (e.g. user:profile1|profile2,user2:profile3)
  -c string
      [health] URL to check
  -csp string
      [owasp] Content-Security-Policy (default "default-src 'self'")
  -directory string
      Directory to serve (default "/data/")
  -hsts
      [owasp] Indicate Strict Transport Security (default true)
  -port string
      Listening port (default "1080")
  -prometheusMetricsHost string
      [prometheus] Allowed hostname to call metrics endpoint (default "localhost")
  -prometheusMetricsPath string
      [prometheus] Metrics endpoint path (default "/metrics")
  -prometheusPrefix string
      [prometheus] Prefix (default "http")
  -rateCount uint
      [rate] IP limit (default 5000)
  -staticURL string
      Static Server URL (default "https://fibr-static.vibioh.fr")
  -tls
      Serve TLS content (default true)
  -tlsCert string
      [tls] PEM Certificate file
  -tlsHosts string
      [tls] Self-signed certificate hosts, comma separated (default "localhost")
  -tlsKey string
      [tls] PEM Key file
  -version string
      Version (used mainly as a cache-buster)
```
