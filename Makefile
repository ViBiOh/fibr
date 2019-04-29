SHELL = /bin/sh

APP_NAME ?= fibr
VERSION ?= $(shell git rev-parse --short HEAD)
AUTHOR ?= $(shell git log --pretty=format:'%an' -n 1)

PACKAGES ?= ./...
APP_PACKAGES = $(shell go list -e $(PACKAGES) | grep -v vendor | grep -v node_modules)

GOBIN=bin
BINARY_PATH=$(GOBIN)/$(APP_NAME)

SERVER_SOURCE = cmd/fibr.go
SERVER_RUNNER = go run $(SERVER_SOURCE)
ifeq ($(DEBUG), true)
	SERVER_RUNNER = dlv debug $(SERVER_SOURCE) --
endif

help: Makefile
	@sed -n 's|^##||p' $< | column -t -s ':' | sed -e 's|^| |'

## $(APP_NAME): Build app with dependencies download
.PHONY: $(APP_NAME)
$(APP_NAME): deps go

## go: Build Golang app
.PHONY: go
go: format lint test bench build

## name: Output name of app
.PHONY: name
name:
	@echo -n $(APP_NAME)

## dist: Output build output path
.PHONY: dist
dist:
	@echo -n $(BINARY_PATH)

## version: Output sha1 of last commit
.PHONY: version
version:
	@echo -n $(VERSION)

## author: Output author's name of last commit
.PHONY: author
author:
	@python -c 'import sys; import urllib; sys.stdout.write(urllib.quote_plus(sys.argv[1]))' "$(AUTHOR)"

## deps: Download dependencies
.PHONY: deps
deps:
	go get github.com/golang/dep/cmd/dep
	go get github.com/kisielk/errcheck
	go get golang.org/x/lint/golint
	go get golang.org/x/tools/cmd/goimports
	dep ensure

## format: Format code of app
.PHONY: format
format:
	goimports -w **/*.go
	gofmt -s -w **/*.go

## lint: Lint code of app
.PHONY: lint
lint:
	golint $(APP_PACKAGES)
	errcheck -ignoretests $(APP_PACKAGES)
	go vet $(APP_PACKAGES)

## test: Test code of app with coverage
.PHONY: test
test:
	script/coverage

## bench: Benchmark code of app
.PHONY: bench
bench:
	go test $(APP_PACKAGES) -bench . -benchmem -run Benchmark.*

## build: Build binary of app
.PHONY: build
build:
	CGO_ENABLED=0 go build -ldflags="-s -w" -installsuffix nocgo -o $(BINARY_PATH) $(SERVER_SOURCE)

## start-deps: Download start dependencies
.PHONY: start-deps
start-deps:
	go get github.com/ViBiOh/auth/cmd/bcrypt

## start: Start app
.PHONY: start
start:
	$(SERVER_RUNNER) \
		-fsDirectory "$(pwd)" \
		-publicURL "http://localhost:1080" \
		-authUsers "admin:admin" \
		-basicUsers "1:admin:$(bcrypt admin)" \
		-frameOptions "SAMEORIGIN" \
		-thumbnailImaginaryURL "" \
		-csp "default-src 'self'; base-uri 'self'; script-src 'unsafe-inline'; style-src 'unsafe-inline'; img-src 'self' data:"
