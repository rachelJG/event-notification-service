.PHONY: lint test test-integration migrate

GOLANGCI_LINT_CACHE ?= /tmp/golangci-lint
GOCACHE ?= /tmp/go-build
GOMODCACHE ?= /tmp/go-mod
GOPATH ?= /tmp/go

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
	bash -c 'set -a; [ -f .env ] && . .env; set +a; go run ./cmd/migrate'
