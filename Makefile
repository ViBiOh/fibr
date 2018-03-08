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
	CGO_ENABLED=0 GOOS=linux GOARCH=arm GOARM=7 go build -ldflags="-s -w" -installsuffix nocgo -o bin/fibr-arm fibr.go
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -ldflags="-s -w" -installsuffix nocgo -o bin/fibr-arm64 fibr.go

docker-deps:
	curl -s -o cacert.pem https://curl.haxx.se/ca/cacert.pem

docker-build:
	docker build -t ${DOCKER_USER}/fibr .
	docker build -t ${DOCKER_USER}/fibr:arm -f Dockerfile_arm .
	docker build -t ${DOCKER_USER}/fibr:arm64 -f Dockerfile_arm64 .

docker-push:
	docker login -u ${DOCKER_USER} -p ${DOCKER_PASS}
	docker push ${DOCKER_USER}/fibr
	docker push ${DOCKER_USER}/fibr:arm
	docker push ${DOCKER_USER}/fibr:arm64

start-deps:
	go get -u github.com/ViBiOh/viws
	go get -u github.com/ViBiOh/auth/bcrypt

start-api:
	go run fibr.go \
		-tls=false \
		-directory `pwd` \
		-publicURL http://localhost:1080 \
		-authUsers admin:admin \
		-basicUsers 1:admin:`bcrypt admin` \
		-csp "default-src 'self'; script-src 'unsafe-inline'; style-src 'unsafe-inline'; img-src 'self'"
