MAKEFLAGS += --silent
GOBIN=bin
BINARY_PATH=$(GOBIN)/$(APP_NAME)
VERSION ?= $(shell git log --pretty=format:'%h' -n 1)
AUTHOR ?= $(shell git log --pretty=format:'%an' -n 1)

APP_NAME ?= fibr

help: Makefile
	@sed -n 's|^##||p' $< | column -t -s ':' | sed -e 's|^| |'

## $(APP_NAME): Build app with dependencies download
$(APP_NAME): deps go

go: format lint tst bench build

## name: Output name of app
name:
	@echo -n $(APP_NAME)

## dist: Output build output path
dist:
	@echo -n $(BINARY_PATH)

## version: Output sha1 of last commit
version:
	@echo -n $(VERSION)

## author: Output author's name of last commit
author:
	@python -c 'import sys; import urllib; sys.stdout.write(urllib.quote_plus(sys.argv[1]))' "$(AUTHOR)"

## deps: Download dependencies
deps:
	go get github.com/golang/dep/cmd/dep
	go get github.com/golang/lint/golint
	go get github.com/kisielk/errcheck
	go get golang.org/x/tools/cmd/goimports
	dep ensure

## format: Format code of app
format:
	goimports -w **/*.go
	gofmt -s -w **/*.go

## lint: Lint code of app
lint:
	golint `go list ./... | grep -v vendor`
	errcheck -ignoretests `go list ./... | grep -v vendor`
	go vet ./...

## tst: Test code of app with coverage
tst:
	script/coverage

## bench: Benchmark code of app
bench:
	go test ./... -bench . -benchmem -run Benchmark.*

## build: Build binary of app
build:
	CGO_ENABLED=0 go build -ldflags="-s -w" -installsuffix nocgo -o $(BINARY_PATH) cmd/fibr.go

start-deps:
	go get github.com/ViBiOh/auth/cmd/bcrypt

## start: Start app
start:
	DEBUG=true go run cmd/fibr.go \
		-tls=false \
		-fsDirectory `pwd` \
		-publicURL http://localhost:1080 \
		-authUsers admin:admin \
		-basicUsers 1:admin:`bcrypt admin` \
		-frameOptions "SAMEORIGIN" \
		-csp "default-src 'self'; base-uri 'self'; script-src 'unsafe-inline'; style-src 'unsafe-inline'; img-src 'self' data:"

.PHONY: help $(APP_NAME) go name dist version author deps format lint tst bench build start-deps start
