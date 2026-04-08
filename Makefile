.PHONY: build build-worker lint test test-race bench bench-validate-event pprof-validate-event-cpu pprof-validate-event-mem test-integration migrate migrate-down docs run-worker

GOLANGCI_LINT_CACHE ?= /tmp/golangci-lint
GOCACHE ?= /tmp/go-build
GOTOOLCHAIN ?= go1.25.8
GOROOT ?= $(shell go env GOROOT)
GOMODCACHE ?= $(shell $(GOROOT)/bin/go env GOMODCACHE)
GOPATH ?= $(shell $(GOROOT)/bin/go env GOPATH)
GO ?= $(GOROOT)/bin/go

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
LDFLAGS := -ldflags="-X main.Version=$(VERSION) -X main.Commit=$(COMMIT)"

build:
	GOCACHE=$(GOCACHE) \
	GOMODCACHE=$(GOMODCACHE) \
	GOPATH=$(GOPATH) \
	GOTOOLCHAIN=$(GOTOOLCHAIN) \
	$(GO) build $(LDFLAGS) -o bin/event-service ./cmd/api

build-worker:
	GOCACHE=$(GOCACHE) \
	GOMODCACHE=$(GOMODCACHE) \
	GOPATH=$(GOPATH) \
	GOTOOLCHAIN=$(GOTOOLCHAIN) \
	$(GO) build $(LDFLAGS) -o bin/worker ./cmd/worker

run-worker:
	GOCACHE=$(GOCACHE) \
	GOMODCACHE=$(GOMODCACHE) \
	GOPATH=$(GOPATH) \
	GOTOOLCHAIN=$(GOTOOLCHAIN) \
	$(GO) run ./cmd/worker

lint:
	GOLANGCI_LINT_CACHE=$(GOLANGCI_LINT_CACHE) \
	GOCACHE=$(GOCACHE) \
	GOMODCACHE=$(GOMODCACHE) \
	GOPATH=$(GOPATH) \
	GOTOOLCHAIN=$(GOTOOLCHAIN) \
	golangci-lint run ./...

test:
	GOCACHE=$(GOCACHE) \
	GOMODCACHE=$(GOMODCACHE) \
	GOPATH=$(GOPATH) \
	GOTOOLCHAIN=$(GOTOOLCHAIN) \
	$(GO) test ./...

test-race:
	GOCACHE=$(GOCACHE) \
	GOMODCACHE=$(GOMODCACHE) \
	GOPATH=$(GOPATH) \
	GOTOOLCHAIN=$(GOTOOLCHAIN) \
	$(GO) test -race ./internal/...

bench:
	GOCACHE=$(GOCACHE) \
	GOMODCACHE=$(GOMODCACHE) \
	GOPATH=$(GOPATH) \
	GOTOOLCHAIN=$(GOTOOLCHAIN) \
	$(GO) test -run=^$$ -bench=. -benchmem ./internal/...

bench-validate-event:
	GOCACHE=$(GOCACHE) \
	GOMODCACHE=$(GOMODCACHE) \
	GOPATH=$(GOPATH) \
	GOTOOLCHAIN=$(GOTOOLCHAIN) \
	$(GO) test -run=^$$ -bench=BenchmarkValidateEvent -benchmem ./internal/application/validation

pprof-validate-event-cpu:
	mkdir -p profiles
	GOCACHE=$(GOCACHE) \
	GOMODCACHE=$(GOMODCACHE) \
	GOPATH=$(GOPATH) \
	GOTOOLCHAIN=$(GOTOOLCHAIN) \
	$(GO) test -run=^$$ -bench=BenchmarkValidateEvent -cpuprofile=profiles/validate_event_cpu.prof ./internal/application/validation

pprof-validate-event-mem:
	mkdir -p profiles
	GOCACHE=$(GOCACHE) \
	GOMODCACHE=$(GOMODCACHE) \
	GOPATH=$(GOPATH) \
	GOTOOLCHAIN=$(GOTOOLCHAIN) \
	$(GO) test -run=^$$ -bench=BenchmarkValidateEvent -memprofile=profiles/validate_event_mem.prof ./internal/application/validation

test-integration:
	GOCACHE=$(GOCACHE) \
	GOMODCACHE=$(GOMODCACHE) \
	GOPATH=$(GOPATH) \
	GOTOOLCHAIN=$(GOTOOLCHAIN) \
	bash -c 'set -a; [ -f .env ] && . .env; set +a; $(GO) test -tags=integration ./...'

migrate:
	GOCACHE=$(GOCACHE) \
	GOMODCACHE=$(GOMODCACHE) \
	GOPATH=$(GOPATH) \
	GOTOOLCHAIN=$(GOTOOLCHAIN) \
	bash -c 'set -a; [ -f .env ] && . .env; set +a; $(GO) run ./cmd/migrate up'

migrate-down:
	GOCACHE=$(GOCACHE) \
	GOMODCACHE=$(GOMODCACHE) \
	GOPATH=$(GOPATH) \
	GOTOOLCHAIN=$(GOTOOLCHAIN) \
	bash -c 'set -a; [ -f .env ] && . .env; set +a; $(GO) run ./cmd/migrate down'

docs:
	swag init -g cmd/api/main.go -o docs --parseDependency --parseInternal
