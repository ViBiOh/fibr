SHELL = /usr/bin/env bash -o nounset -o pipefail -o errexit -c

ifneq ("$(wildcard .env)","")
	include .env
	export
endif

APP_NAME = fibr
PACKAGES ?= ./...

MAIN_SOURCE = cmd/fibr/fibr.go
MAIN_RUNNER = go run $(MAIN_SOURCE)
ifeq ($(DEBUG), true)
	MAIN_RUNNER = dlv debug $(MAIN_SOURCE) --
endif

.DEFAULT_GOAL := app

## help: Display list of commands
.PHONY: help
help: Makefile
	@sed -n 's|^##||p' $< | column -t -s ':' | sort

## name: Output app name
.PHONY: name
name:
	@printf "$(APP_NAME)"

## version: Output last commit sha1
.PHONY: version
version:
	@printf "$(shell git rev-parse --short HEAD)"

## dev: Build app
.PHONY: dev
dev: format style test build

## app: Build whole app
.PHONY: app
app: init dev

## init: Bootstrap your application. e.g. fetch some data files, make some API calls, request user input etc...
.PHONY: init
init:
	@curl --disable --silent --show-error --location --max-time 30 "https://raw.githubusercontent.com/ViBiOh/scripts/main/bootstrap" | bash -s -- "-c" "git_hooks" "coverage" "release"
	go install "github.com/golang/mock/mockgen@v1.6.0"
	go install "github.com/kisielk/errcheck@latest"
	go install "golang.org/x/lint/golint@latest"
	go install "golang.org/x/tools/cmd/goimports@latest"
	go install "mvdan.cc/gofumpt@latest"
	go mod tidy -compat=1.17

## format: Format code. e.g Prettier (js), format (golang)
.PHONY: format
format:
	goimports -w $(shell find . -name "*.go")
	gofumpt -w $(shell find . -name "*.go") 2>/dev/null

## style: Check lint, code styling rules. e.g. pylint, phpcs, eslint, style (java) etc ...
.PHONY: style
style:
	golint $(PACKAGES)
	errcheck -ignoretests $(PACKAGES)
	go vet $(PACKAGES)

## mocks: Generate mocks
.PHONY: mocks
mocks:
	find . -name "mocks" -type d -exec rm -r "{}" \+
	go generate $(PACKAGES)

## test: Shortcut to launch all the test tasks (unit, functional and integration).
.PHONY: test
test:
	scripts/coverage
	$(MAKE) bench

## bench: Shortcut to launch benchmark tests.
.PHONY: bench
bench:
	go test $(PACKAGES) -bench . -benchmem -run Benchmark.*

## build: Build the application.
.PHONY: build
build:
	CGO_ENABLED=0 go build -ldflags="-s -w" -installsuffix nocgo -o bin/$(APP_NAME) $(MAIN_SOURCE)

## run: Locally run the application, e.g. node index.js, python -m myapp, go run myapp etc ...
.PHONY: run
run:
	$(MAIN_RUNNER) \
		-ignorePattern ".git" \
		-authUsers "1:`htpasswd -nBb admin admin`"
