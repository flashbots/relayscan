# Heavily inspired by Lighthouse: https://github.com/sigp/lighthouse/blob/stable/Makefile
# and Reth: https://github.com/paradigmxyz/reth/blob/main/Makefile
.DEFAULT_GOAL := help

VERSION := $(shell git describe --tags --always --dirty="-dev")

##@ Help

help: ## Display this help
	@awk 'BEGIN {FS = ":.*##"; printf "Usage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

v: ## Show the current version
	@echo "Version: ${VERSION}"

##@ Building

clean: ## Remove build artifacts
	rm -rf relayscan build/

build: ## Build the relayscan binary
	go build -trimpath -ldflags "-s -X cmd.Version=${VERSION} -X main.Version=${VERSION}" -v -o relayscan .

docker-image: ## Build the relayscan docker image
	DOCKER_BUILDKIT=1 docker build --platform linux/amd64 --build-arg VERSION=${VERSION} . -t relayscan

generate-ssz: ## Generate SSZ serialization code
	rm -f common/ultrasoundbid_encoding.go
	sszgen --path common --objs UltrasoundStreamBid

##@ Production tasks

update-bids-website: ## Update the bid archive website
	go run . service bidcollect --build-website --build-website-upload

##@ Linting and Testing

lint: ## Lint the code
	gofmt -d -s .
	gofumpt -d -extra .
	go vet ./...
	staticcheck ./...
	golangci-lint run

test: ## Run tests
	go test ./...

test-race: ## Run tests with -race fla
	go test -race ./...

lt: lint test ## Run lint and tests

gofumpt: ## Run gofumpt on the code
	gofumpt -l -w -extra .

fmt: ## Format the code with gofmt and gofumpt and gc
	gofmt -s -w .
	gofumpt -extra -w .
	gci write .
	go mod tidy

cover: ## Run tests with coverage
	go test -coverprofile=/tmp/go-sim-lb.cover.tmp ./...
	go tool cover -func /tmp/go-sim-lb.cover.tmp
	unlink /tmp/go-sim-lb.cover.tmp

cover-html: ## Run tests with coverage and output the HTML report
	go test -coverprofile=/tmp/go-sim-lb.cover.tmp ./...
	go tool cover -html=/tmp/go-sim-lb.cover.tmp
	unlink /tmp/go-sim-lb.cover.tmp

##@ Development

dev-website: ## Run the relayscan website service in development mode
	DB_DONT_APPLY_SCHEMA=1 go run . service website --dev

dev-bids-website: ## Run the bidcollect website in development mode
	go run . service bidcollect --devserver

dev-postgres-start: ## Start a Postgres container for development
	docker run -d --name relayscan-postgres -p 5432:5432 -e POSTGRES_USER=postgres -e POSTGRES_PASSWORD=postgres -e POSTGRES_DB=postgres postgres

dev-postgres-stop: ## Stop the Postgres container
	docker rm -f relayscan-postgres

dev-postgres-wipe: dev-postgres-stop dev-postgres-start ## Restart the Postgres container (wipes the database)
