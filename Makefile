default: deps dev docker

dev: format lint tst bench build

deps:
	go get -u github.com/golang/lint/golint
	go get -u github.com/NYTimes/gziphandler
	go get -u github.com/ViBiOh/alcotest/alcotest
	go get -u github.com/ViBiOh/auth/auth
	go get -u github.com/ViBiOh/httputils
	go get -u github.com/ViBiOh/httputils/cert
	go get -u github.com/ViBiOh/httputils/owasp
	go get -u github.com/ViBiOh/httputils/prometheus
	go get -u github.com/ViBiOh/httputils/rate
	go get -u golang.org/x/tools/cmd/goimports

format:
	goimports -w *.go
	gofmt -s -w *.go

lint:
	golint ./...
	go vet ./...

tst:
	script/coverage

bench:
	go test ./... -bench . -benchmem -run Benchmark.*

build:
	CGO_ENABLED=0 go build -ldflags="-s -w" -installsuffix nocgo -o bin/fibr fibr.go

docker: docker-deps docker-build

docker-deps:
	curl -s -o cacert.pem https://curl.haxx.se/ca/cacert.pem

docker-build:
	docker build -t ${DOCKER_USER}/fibr .