default: go docker

go: deps dev

dev: format lint tst bench build

docker: docker-deps docker-build

deps:
	go get -u github.com/golang/dep/cmd/dep
	go get -u github.com/golang/lint/golint
	go get -u golang.org/x/tools/cmd/goimports
	dep ensure

format:
	goimports -w *.go
	gofmt -s -w *.go

lint:
	golint `go list ./... | grep -v vendor`
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

docker-push:
	docker login -u ${DOCKER_USER} -p ${DOCKER_PASS}
	docker push ${DOCKER_USER}/fibr

start-deps:
	go get -u github.com/ViBiOh/auth
	go get -u github.com/ViBiOh/auth/bcrypt

start-auth:
	auth \
	  -tls=false \
	  -basicUsers "1:admin:`bcrypt admin`" \
	  -corsHeaders Content-Type,Authorization \
	  -port 1081 \
	  -corsCredentials

start-api:
	go run fibr.go \
	  -tls=false \
	  -directory `pwd` \
	  -authUrl http://localhost:1081 \
	  -authUsers admin:admin \
	  -csp "default-src 'self'; style-src 'self' 'unsafe-inline'"
