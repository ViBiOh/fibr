default: go docker

go: deps dev

dev: format lint tst bench build

docker: docker-deps docker-build

deps:
	go get -u github.com/golang/dep/cmd/dep
	go get -u github.com/golang/lint/golint
	go get -u github.com/kisielk/errcheck
	go get -u golang.org/x/tools/cmd/goimports
	dep ensure

format:
	goimports -w *.go
	gofmt -s -w *.go

lint:
	golint `go list ./... | grep -v vendor`
	errcheck -ignoretests `go list ./... | grep -v vendor`
	go vet ./...

tst:
	script/coverage

bench:
	go test ./... -bench . -benchmem -run Benchmark.*

build:
	CGO_ENABLED=0 go build -ldflags="-s -w" -installsuffix nocgo -o bin/fibr fibr.go

docker-deps:
	curl -s -o cacert.pem https://curl.haxx.se/ca/cacert.pem

docker-build:
	docker build -t ${DOCKER_USER}/fibr .
	docker build -t ${DOCKER_USER}/fibr-static -f Dockerfile_static .

docker-push:
	docker login -u ${DOCKER_USER} -p ${DOCKER_PASS}
	docker push ${DOCKER_USER}/fibr
	docker push ${DOCKER_USER}/fibr-static

start-deps:
	go get -u github.com/ViBiOh/viws
	go get -u github.com/ViBiOh/auth/bcrypt

start-static:
	viws \
		-directory `pwd`/web/static/ \
		-port 1081

start-api:
	go run fibr.go \
		-tls=false \
		-directory `pwd` \
		-staticURL http://localhost:1081 \
		-publicURL http://localhost:1080 \
		-authUsers admin:admin \
		-basicUsers 1:admin:`bcrypt password` \
		-csp "default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'; img-src 'self' fibr-static.vibioh.fr"
