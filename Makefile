SHELL = /bin/bash

ifneq ("$(wildcard .env)","")
	include .env
	export
endif

APP_NAME = fibr
PACKAGES ?= ./...

MAIN_SOURCE = cmd/fibr/fibr.go
MAIN_RUNNER = go run $(MAIN_SOURCE)
ifeq ($(DEBUG), true)
	MAIN_RUNNER = gdlv -d $(shell dirname $(MAIN_SOURCE)) debug --
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
	go install github.com/kisielk/errcheck@latest
	go install golang.org/x/lint/golint@latest
	go install golang.org/x/tools/cmd/goimports@latest
	go install github.com/golang/mock/mockgen@v1.6.0
	$(MAKE) mocks
	go mod tidy

## format: Format code. e.g Prettier (js), format (golang)
.PHONY: format
format:
	goimports -w $(shell find . -name "*.go")
	gofmt -s -w $(shell find . -name "*.go")

## style: Check lint, code styling rules. e.g. pylint, phpcs, eslint, style (java) etc ...
.PHONY: style
style:
	golint $(PACKAGES)
	errcheck -ignoretests $(PACKAGES)
	go vet $(PACKAGES)

## mocks: Generate mocks
.PHONY: mocks
mocks:
	find . -name "mock" -type d -exec rm {} \;
	mockgen -destination pkg/mocks/crud.go -mock_names App=Crud -package mocks github.com/ViBiOh/fibr/pkg/crud App
	mockgen -destination pkg/mocks/share.go -mock_names App=Share -package mocks github.com/ViBiOh/fibr/pkg/share App
	mockgen -destination pkg/mocks/storage.go -mock_names Storage=Storage -package mocks github.com/ViBiOh/fibr/pkg/provider Storage
	mockgen -destination pkg/mocks/auth_middleware.go -mock_names App=AuthMiddleware -package mocks github.com/ViBiOh/auth/v2/pkg/middleware App

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
		-fsDirectory "$(PWD)" \
		-ignorePattern ".git" \
		-publicURL "http://localhost:1080" \
		-authProfiles "1:admin" \
		-authUsers "1:`htpasswd -nBb admin admin`" \
		-frameOptions "SAMEORIGIN" \
		-thumbnailImageURL "https://imaginary.vibioh.fr" \
		-thumbnailVideoURL "https://vith.vibioh.fr" \
		-exifURL "http://localhost:4000" \
		-exifGeocodeURL "https://nominatim.openstreetmap.org" \
		-exifDateOnStart \
		-exifAggregateOnStart

.PHONY: run-imaginary
run-imaginary:
	docker run --rm \
		--name "imaginary" \
		-p "9000:9000/tcp" \
		"h2non/imaginary"

.PHONY: run-vith
run-vith:
	docker run --rm \
		--name "vith" \
		-p "2080:1080/tcp" \
		"vibioh/vith"
