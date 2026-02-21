.PHONY: build lint test test-integration migrate migrate-down docs

GOLANGCI_LINT_CACHE ?= /tmp/golangci-lint
GOCACHE ?= /tmp/go-build
GOMODCACHE ?= /tmp/go-mod
GOPATH ?= /tmp/go

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
LDFLAGS := -ldflags="-X main.Version=$(VERSION) -X main.Commit=$(COMMIT)"

build:
	GOCACHE=$(GOCACHE) \
	GOMODCACHE=$(GOMODCACHE) \
	GOPATH=$(GOPATH) \
	go build $(LDFLAGS) -o event-service ./cmd/api

lint:
	GOLANGCI_LINT_CACHE=$(GOLANGCI_LINT_CACHE) \
	GOCACHE=$(GOCACHE) \
	GOMODCACHE=$(GOMODCACHE) \
	GOPATH=$(GOPATH) \
	golangci-lint run ./...

test:
	GOCACHE=$(GOCACHE) \
	GOMODCACHE=$(GOMODCACHE) \
	GOPATH=$(GOPATH) \
	go test ./...

test-integration:
	GOCACHE=$(GOCACHE) \
	GOMODCACHE=$(GOMODCACHE) \
	GOPATH=$(GOPATH) \
	bash -c 'set -a; [ -f .env ] && . .env; set +a; go test -tags=integration ./...'

migrate:
	GOCACHE=$(GOCACHE) \
	GOMODCACHE=$(GOMODCACHE) \
	GOPATH=$(GOPATH) \
	bash -c 'set -a; [ -f .env ] && . .env; set +a; go run ./cmd/migrate up'

migrate-down:
	GOCACHE=$(GOCACHE) \
	GOMODCACHE=$(GOMODCACHE) \
	GOPATH=$(GOPATH) \
	bash -c 'set -a; [ -f .env ] && . .env; set +a; go run ./cmd/migrate down'

docs:
	npx --yes aglio -i docs/api.apib -o docs/api.html
