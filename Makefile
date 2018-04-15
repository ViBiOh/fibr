VERSION ?= $(shell git log --pretty=format:'%h' -n 1)

default: api docker

api: deps go

go: format lint tst bench build

docker: docker-build docker-push

version:
	@echo -n $(VERSION)

deps:
	go get -u github.com/golang/dep/cmd/dep
	go get -u github.com/golang/lint/golint
	go get -u github.com/kisielk/errcheck
	go get -u golang.org/x/tools/cmd/goimports
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
	CGO_ENABLED=0 go build -ldflags="-s -w" -installsuffix nocgo -o bin/fibr cmd/fibr.go

docker-deps:
	curl -s -o cacert.pem https://curl.haxx.se/ca/cacert.pem

docker-login:
	echo $(DOCKER_PASS) | docker login -u $(DOCKER_USER) --password-stdin

docker-build: docker-deps
	docker build -t $(DOCKER_USER)/fibr:$(VERSION) .

docker-push: docker-login
	docker push $(DOCKER_USER)/fibr:$(VERSION)

docker-pull:
	docker pull $(DOCKER_USER)/fibr:$(VERSION)

docker-promote: docker-pull
	docker tag $(DOCKER_USER)/fibr:$(VERSION) $(DOCKER_USER)/fibr:latest

start-deps:
	go get -u github.com/ViBiOh/auth/cmd/bcrypt

start-api:
	go run cmd/fibr.go \
		-tls=false \
		-directory `pwd` \
		-publicURL http://localhost:1080 \
		-authUsers admin:admin \
		-basicUsers 1:admin:`bcrypt admin` \
		-csp "default-src 'self'; script-src 'unsafe-inline'; style-src 'unsafe-inline'; img-src 'self' data:"

.PHONY: api go docker version deps format lint tst bench build docker-deps docker-login docker-build docker-push docker-pull docker-promote start-deps start-api
