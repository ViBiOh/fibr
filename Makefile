APP_NAME ?= fibr
VERSION ?= $(shell git log --pretty=format:'%h' -n 1)
AUTHOR ?= $(shell git log --pretty=format:'%an' -n 1)

docker:
	docker build -t vibioh/$(APP_NAME):$(VERSION) .

$(APP_NAME): deps go

go: format lint tst bench build

name:
	@echo -n $(APP_NAME)

version:
	@echo -n $(VERSION)

author:
	@python -c 'import sys; import urllib; sys.stdout.write(urllib.quote_plus(sys.argv[1]))' "$(AUTHOR)"

deps:
	go get github.com/golang/dep/cmd/dep
	go get github.com/golang/lint/golint
	go get github.com/kisielk/errcheck
	go get golang.org/x/tools/cmd/goimports
	dep ensure

format:
	goimports -w **/*.go
	gofmt -s -w **/*.go

lint:
	golint `go list ./... | grep -v vendor`
	errcheck -ignoretests `go list ./... | grep -v vendor`
	go vet ./...

tst:
	script/coverage

bench:
	go test ./... -bench . -benchmem -run Benchmark.*

build:
	CGO_ENABLED=0 go build -ldflags="-s -w" -installsuffix nocgo -o bin/$(APP_NAME) cmd/fibr.go

start-deps:
	go get github.com/ViBiOh/auth/cmd/bcrypt

start:
	DEBUG=true go run cmd/fibr.go \
		-tls=false \
		-fsDirectory `pwd` \
		-publicURL http://localhost:1080 \
		-authUsers admin:admin \
		-basicUsers 1:admin:`bcrypt admin` \
		-frameOptions "SAMEORIGIN" \
		-csp "default-src 'self'; base-uri 'self'; script-src 'unsafe-inline'; style-src 'unsafe-inline'; img-src 'self' data:"

.PHONY: docker $(APP_NAME) go name version author deps format lint tst bench build start-deps start
