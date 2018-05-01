# fibr

Web File Browser and Manager

[![Build Status](https://travis-ci.org/ViBiOh/fibr.svg?branch=master)](https://travis-ci.org/ViBiOh/fibr)
[![Go Report Card](https://goreportcard.com/badge/github.com/ViBiOh/fibr)](https://goreportcard.com/report/github.com/ViBiOh/fibr)

Thanks to [FontAwesome](https://fontawesome.com) for providing awesome svg.

## Installation

```bash
go get -u github.com/ViBiOh/fibr/cmd
```

## Usage

```bash
  -authUrl string
      [auth] Auth URL, if remote
  -authUsers string
      [auth] List of allowed users and profiles (e.g. user:profile1|profile2,user2:profile3)
  -basicUsers string
      [Basic] Users in the form "id:username:password,id2:username2:password2"
  -csp string
      [owasp] Content-Security-Policy (default "default-src 'self'; base-uri 'self'")
  -datadogHostname string
      Datadog Agent Hostname (default "dd-agent")
  -datadogPort string
      Datadog Agent Port (default "8126")
  -datadogService string
      Service name
  -directory string
      Directory to serve (default "/data")
  -frameOptions string
      [owasp] X-Frame-Options (default "deny")
  -hsts
      [owasp] Indicate Strict Transport Security (default true)
  -metadata
      Enable metadata storage (default true)
  -port int
      Listen port (default 1080)
  -publicURL string
      [fibr] Public URL (default "https://fibr.vibioh.fr")
  -tls
      Serve TLS content (default true)
  -tlsCert string
      [tls] PEM Certificate file
  -tlsHosts string
      [tls] Self-signed certificate hosts, comma separated (default "localhost")
  -tlsKey string
      [tls] PEM Key file
  -url string
      [health] URL to check
  -version string
      [fibr] Version (used mainly as a cache-buster)
```
