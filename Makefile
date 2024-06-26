VERSION := $(shell git describe --tags --always --dirty="-dev")

all: build-portable

v:
	@echo "Version: ${VERSION}"

clean:
	rm -rf relayscan build/

build:
	go build -trimpath -ldflags "-s -X cmd.Version=${VERSION} -X main.Version=${VERSION}" -v -o relayscan .

build-portable:
	go build -trimpath -ldflags "-s -X cmd.Version=${VERSION} -X main.Version=${VERSION}" -v -o relayscan .

test:
	go test ./...

test-race:
	go test -race ./...

lint:
	gofmt -d -s .
	gofumpt -d -extra .
	go vet ./...
	staticcheck ./...
	golangci-lint run

lt: lint test

gofumpt:
	gofumpt -l -w -extra .

fmt:
	gofmt -s -w .
	gofumpt -extra -w .
	gci write .
	go mod tidy

cover:
	go test -coverprofile=/tmp/go-sim-lb.cover.tmp ./...
	go tool cover -func /tmp/go-sim-lb.cover.tmp
	unlink /tmp/go-sim-lb.cover.tmp

cover-html:
	go test -coverprofile=/tmp/go-sim-lb.cover.tmp ./...
	go tool cover -html=/tmp/go-sim-lb.cover.tmp
	unlink /tmp/go-sim-lb.cover.tmp

docker-image:
	DOCKER_BUILDKIT=1 docker build --platform linux/amd64 --build-arg VERSION=${VERSION} . -t relayscan

generate-ssz:
	rm -f common/top-bid-ws-bid_encoding.go
	sszgen --path common --objs TopBidWebsocketStreamBid

update-bids-website:
	go run . service bidcollect --build-website --build-website-upload

dev-website:
	go run . service website --dev

dev-bids-website:
	go run . service bidcollect --devserver
