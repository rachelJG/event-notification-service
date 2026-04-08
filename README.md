# Event Notification Service

A production-ready, high-performance event notification service built in Go with hexagonal architecture. Receives domain events via REST API, persists them in PostgreSQL, and processes them asynchronously through a worker to deliver notifications via email and WhatsApp.

[![CI](https://github.com/rachelJG/event-notification-service/workflows/CI/badge.svg)](https://github.com/rachelJG/event-notification-service/actions)
[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go)](https://go.dev/)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

---

## 📢 Recent Updates

### March 2026

- **🔍 OpenTelemetry Tracing**: Added distributed tracing with automatic HTTP instrumentation, use case spans, and infrastructure-level tracing. Exporters support stdout (dev) and OTLP (prod via Tempo/Jaeger).
- **🎯 Simplified Architecture**: Removed application-level rate limiting (commit `318e50f`) in favor of infrastructure-level rate limiting at load balancers/API gateways.
- **🧪 Increased Test Coverage**: Unit test coverage raised from 34% to **≥80%** with comprehensive tests across domain, application, and infrastructure layers.
- **🔧 Centralized Error Handling**: Refactored HTTP error responses into a centralized `ErrorHandler` middleware for consistent error formatting.
- **📊 Observability Stack**: Added full Docker Compose observability stack with Grafana + Prometheus + Tempo + Loki for metrics, traces, and logs correlation.

See the [full changelog](https://github.com/rachelJG/event-notification-service/commits/main) for details.

---

## 🎯 Overview

This service implements a **complete event-driven notification pipeline** with:

- **REST API** for event ingestion with idempotency guarantees
- **Asynchronous Worker** for event processing and notification delivery
- **Multi-channel delivery**: Email (SMTP) and WhatsApp (HTTP API)
- **Fan-out notifications**: A single event can generate multiple notifications (e.g., one invoice event per condominium fans out to N resident emails)
- **Production hardening**: rate limiting, API Key authentication, CORS, security headers, graceful shutdown
- **Client tracking**: Automatic capture of client metadata for auditing and analytics
- **Full observability**: Prometheus metrics, structured logging (Zap), OpenTelemetry distributed tracing, health checks
- **Clean architecture**: Hexagonal/Ports & Adapters pattern with zero domain dependencies

### Key Design Decisions

| Aspect | Choice | Rationale |
|--------|--------|-----------|
| **API Style** | REST (HTTP/JSON) | Simple, wide tooling support, sufficient for event submission |
| **Storage** | PostgreSQL 15 | ACID guarantees, transactional workflows, idempotency via constraints |
| **Processing** | Async worker (polling) | Decouples producers from delivery, resilient to external failures |
| **Architecture** | Hexagonal | Domain isolation, testability, infrastructure independence |
| **Auth** | API Keys (SHA-256) | Long-lived credentials for service-to-service communication with client tracking |

---

## 🏗️ Architecture

### System Components

```
┌──────────────┐                 ┌──────────────┐
│   Producer   │                 │   Consumer   │
│  (External)  │                 │  (External)  │
└──────┬───────┘                 └──────▲───────┘
       │                                │
       │ HTTP POST                      │ Email / WhatsApp
       ▼                                │
┌─────────────────────────────────────────────────┐
│              Event Notification Service         │
│                                                  │
│  ┌────────────┐          ┌──────────────┐      │
│  │  API (Gin) │          │    Worker    │      │
│  │  :8080     │          │              │      │
│  └─────┬──────┘          └──────┬───────┘      │
│        │                        │               │
│        │ Persist                │ Poll every 5s │
│        ▼                        ▼               │
│  ┌────────────────────────────────────┐        │
│  │         PostgreSQL 15              │        │
│  │  events table + notifications table│        │
│  └────────────────────────────────────┘        │
│                                                  │
│  Prometheus Metrics: :8080/metrics (:9090)     │
└──────────────────────────────────────────────────┘
```

### Hexagonal Architecture

The project follows **Hexagonal Architecture** (Ports & Adapters) organized in three layers:

```
                         ┌─────────────────────────────────────────┐
                         │         cmd/api/main.go                 │
                         │         cmd/worker/main.go              │
                         │        (Composition Roots)              │
                         └────────────────┬────────────────────────┘
                                          │ wires dependencies
                  ┌───────────────────────┼───────────────────────┐
                  │                       │                       │
                  ▼                       ▼                       ▼
   ┌──────────────────────┐ ┌──────────────────────┐ ┌────────────────────┐
   │   Infrastructure     │ │     Application      │ │      Domain        │
   │                      │ │                      │ │                    │
   │  http/               │ │  usecases/           │ │  entities/         │
   │   ├─ handler.go      │ │   ├─ event_service  │ │   ├─ event.go      │
   │   ├─ server.go       │ │   ├─ process_events │ │   └─ notification  │
   │   ├─ middleware/     │ │   └─ deliver_        │ │                    │
   │   └─ metrics.go      │ │      notifications   │ │  ports/            │
   │                      │ │                      │ │   ├─ event_        │
   │  postgres/           │ │  ports/              │ │   │  repository.go │
   │   ├─ event_repo.go   │ │   ├─ event_service  │ │   ├─ notification_ │
   │   ├─ notification_   │ │   ├─ event_         │ │   │  repository.go │
   │   │  repo.go         │ │   │  processor      │ │   ├─ email_sender  │
   │   └─ metrics.go      │ │   └─ notification_  │ │   └─ email_        │
   │                      │ │      deliverer       │ │      renderer.go   │
   │  email/              │ │                      │ │   ├─ whatsapp_     │
   │   ├─ smtp_adapter.go │ │  validation/         │ │   │  sender.go     │
   │   └─ templates.go    │ │   └─ validate_       │ │   └─ email_        │
   │                      │ │      event.go        │ │      renderer.go   │
   │  whatsapp/           │ │                      │ │                    │
   │   └─ sender.go       │ │                      │ │                    │
   │                      │ │                      │ │                    │
   │  logger/             │ │                      │ │                    │
   │   └─ zap.go          │ │                      │ │                    │
   └──────────┬───────────┘ └──────────┬───────────┘ └────────────────────┘
              │                        │                       ▲
              │         implements     │        depends on     │
              └────────────────────────┴───────────────────────┘
                              ports (interfaces)
```

**Dependency Rule**: Dependencies point inward. Infrastructure and Application depend on Domain (ports), but Domain never imports from outer layers.

### Directory Structure

```
event-notification-service/
├── cmd/
│   ├── api/main.go                    # API server entry point
│   └── worker/
│       ├── main.go                    # Worker entry point
│       └── metrics.go                 # Worker-specific metrics
├── internal/
│   ├── config/                        # Environment-based configuration
│   ├── domain/
│   │   ├── entities/                  # Event, Notification, EventType
│   │   └── ports/                     # Output ports (repositories, senders)
│   ├── application/
│   │   ├── ports/                     # Input ports (use case interfaces)
│   │   ├── usecases/                  # Business logic
│   │   └── validation/                # Payload validation per event type
│   ├── infrastructure/
│   │   ├── http/                      # Gin router, handlers, middleware, metrics
│   │   ├── postgres/                  # Repository implementations, migrations
│   │   ├── email/                     # SMTP adapter, email templates
│   │   ├── whatsapp/                  # WhatsApp API sender
│   │   ├── telemetry/                 # OpenTelemetry tracing initialization
│   │   └── logger/                    # Zap logger factory
│   ├── pkg/
│   │   └── apperror/                  # Typed error taxonomy
│   └── tests/                         # Integration tests (//go:build integration)
├── deploy/
│   └── k8s/                           # Kubernetes manifests
├── docs/
│   └── api.apib                       # API Blueprint documentation
├── .github/workflows/
│   ├── ci.yml                         # Lint, test, coverage (≥80%)
│   └── deploy.yml                     # Migrations → Build → Deploy pipeline
├── observability/
│   ├── prometheus.yml                 # Prometheus scrape config
│   ├── tempo.yml                      # Tempo OTLP receiver config
│   ├── loki.yml                       # Loki log aggregation config
│   └── promtail.yml                   # Promtail log collection config
├── docker-compose.yml                 # Local development setup
├── Makefile                           # Build, test, lint, migrate targets
├── CLAUDE.md                          # Codebase conventions for AI agents
└── AGENT.md                           # Project state tracker (all phases DONE)
```

---

## ✨ Features

### Core Functionality

- ✅ **Event Ingestion**: REST API with API Key authentication and idempotency keys
- ✅ **Supported Event Types**:
  - `UserRegistered` — User registration notifications (email)
  - `PasswordResetRequested` — Password reset emails (email)
  - `OrderPaid` — Payment confirmation (email)
  - `OrderShipped` — Shipping notifications (email)
  - `InvoiceIssued` — Condominium invoice fan-out: 1 event → N individual emails per resident
  - `InvoiceSummary` — WhatsApp group summary notification for condominium administrators
- ✅ **Idempotency**: Guaranteed via `UNIQUE(idempotency_key, type)` DB constraint
- ✅ **Async Processing**: Worker polls pending events every 5 seconds
- ✅ **Multi-channel Delivery**: SMTP for email, HTTP API for WhatsApp
- ✅ **Fan-out**: `InvoiceIssued` events produce one notification per recipient (up to 500)
- ✅ **Retry Logic**: Configurable max retries with exponential backoff

### Production Hardening

- 🔒 **Security**:
  - API Key authentication (SHA-256 hashed, scope-based authorization)
  - Client tracking via metadata (client_id, organization, contact)
  - Security headers (HSTS, X-Content-Type-Options, X-Frame-Options)
  - Trusted proxy configuration
  - **Note**: Rate limiting should be configured at the infrastructure level (load balancer/API gateway)
- 🛡️ **Reliability**:
  - Graceful shutdown (configurable timeout)
  - Context-aware timeouts (DB queries, HTTP requests)
  - Circuit breaker patterns ready
  - Dead letter queue support (status: failed)
- 📊 **Observability**:
  - Prometheus metrics (HTTP, DB pool, business metrics)
  - Structured logging (Zap) with request IDs and client_id tracking
  - **OpenTelemetry distributed tracing** with automatic HTTP instrumentation
  - API Key audit logging (auth success/failure with IP and client_id)
  - Health check endpoints (`/health/live`, `/health/ready` with DB stats)
  - Integrated observability stack (Grafana + Prometheus + Tempo + Loki)

### Metrics Exposed

**API Server** (`:8080/metrics`):
- `http_requests_total{method,path,status}` — HTTP request counter
- `http_request_duration_seconds{method,path}` — Request latency histogram
- `http_errors_total{code}` — HTTP error counter by status code
- `events_submitted_total{event_type,result}` — Business metric for event submission
- `postgres_pool_*` — Database connection pool stats

**Worker** (`:9090/metrics`):
- `events_processed_total{result}` — Events processed (success/error)
- `notifications_delivered_total{channel,result}` — Delivery attempts
- `worker_cycles_total{loop,result}` — Worker loop iterations
- `postgres_pool_*` — Database connection pool stats

---

## 🚀 Getting Started

### Prerequisites

- **Docker & Docker Compose** (recommended for local development)
- **Go 1.21+** (for building from source)
- **PostgreSQL 15** (if running without Docker)
- **Make** (optional, for convenience commands)

### Quick Start with Docker

1. **Clone the repository**:
   ```bash
   git clone https://github.com/rachelJG/event-notification-service.git
   cd event-notification-service
   ```

2. **Configure environment**:
   ```bash
   cp .env.example .env
   # Edit .env with your SMTP credentials and JWT secret
   ```

3. **Start all services**:
   ```bash
   docker-compose up -d
   ```

   This starts:
   - API server on `:8080`
   - Worker (background process)
   - PostgreSQL on `:5432`

4. **Apply migrations**:
   ```bash
   make migrate
   # Or manually: bash ./scripts/run-migrations.sh
   ```

5. **Verify health**:
   ```bash
   curl http://localhost:8080/health
   ```

   Expected response:
   ```json
   {
     "status": "ok",
     "version": "v0.1.0",
     "commit": "abc1234",
     "db": {
       "acquire_count": 0,
       "acquired_conns": 0,
       "idle_conns": 2,
       "max_conns": 10,
       "total_conns": 2
     }
   }
   ```

### Development Setup (Without Docker)

1. **Install dependencies**:
   ```bash
   go mod download
   ```

2. **Start PostgreSQL**:
   ```bash
   # Using Docker:
   docker run -d --name postgres \
     -e POSTGRES_USER=postgres \
     -e POSTGRES_PASSWORD=postgres \
     -e POSTGRES_DB=events \
     -p 5432:5432 \
     postgres:15-alpine
   ```

3. **Apply migrations**:
   ```bash
   export PG_DSN="postgres://postgres:postgres@localhost:5432/events?sslmode=disable"
   make migrate
   ```

4. **Run the API**:
   ```bash
   export JWT_SECRET="your-32-byte-secret-key-here-change-me"
   go run ./cmd/api
   ```

5. **Run the worker** (in another terminal):
   ```bash
   export SMTP_HOST="smtp.gmail.com"
   export SMTP_PORT="587"
   export SMTP_USER="your-email@gmail.com"
   export SMTP_PASSWORD="your-app-password"
   export SMTP_FROM="noreply@example.com"
   go run ./cmd/worker
   ```

---

## 📖 API Documentation

### Interactive Documentation (Swagger UI)

The API provides **interactive Swagger documentation** available when the server is running:

🔗 **[http://localhost:8080/swagger/index.html](http://localhost:8080/swagger/index.html)**

The Swagger UI allows you to:
- Explore all available endpoints with detailed descriptions
- View request/response schemas with examples
- Test API calls directly from your browser
- See authentication requirements for each endpoint

To regenerate the Swagger docs after making changes to handlers or DTOs:
```bash
make docs
```

This generates:
- `docs/swagger.json` - OpenAPI 3.0 specification (JSON format)
- `docs/swagger.yaml` - OpenAPI 3.0 specification (YAML format)
- `docs/docs.go` - Embedded Go documentation

### Endpoints

#### **POST /api/v1/events**
Submit a new event for processing. The `client_id` from your API key's metadata is automatically captured and stored with the event.

**Headers**:
- `X-API-Key: <api-key>` (required, scope: `events:write`)
- `Idempotency-Key: <uuid>` (required)
- `Content-Type: application/json` (required)

**Request Body**:
```json
{
  "event_type": "UserRegistered",
  "payload": {
    "user_id": "12345",
    "email": "user@example.com",
    "name": "John Doe"
  }
}
```

**Responses**:
- `202 Accepted` — Event accepted and queued
  ```json
  {"id": "550e8400-e29b-41d4-a716-446655440000"}
  ```
  Headers: `Location: /api/v1/events/{id}`

- `400 Bad Request` — Invalid input (missing idempotency key, bad JSON, validation error)
- `401 Unauthorized` — Missing or invalid API key
- `409 Conflict` — Duplicate event (idempotent: returns original ID)
- `429 Too Many Requests` — Rate limit exceeded (configure at load balancer level)

**Example**:
```bash
curl -X POST http://localhost:8080/api/v1/events \
  -H "X-API-Key: your-api-key-here" \
  -H "Idempotency-Key: $(uuidgen)" \
  -H "Content-Type: application/json" \
  -d '{
    "event_type": "UserRegistered",
    "payload": {
      "user_id": "12345",
      "email": "user@example.com",
      "name": "John Doe"
    }
  }'
```

#### **GET /api/v1/events/:id**
Retrieve a previously submitted event, including the client_id that submitted it.

**Headers**:
- `X-API-Key: <api-key>` (required, scope: `events:read`)

**Responses**:
- `200 OK`
  ```json
  {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "type": "UserRegistered",
    "payload": {"user_id": "12345", "email": "user@example.com", "name": "John Doe"},
    "client_id": "acme-corp",
    "occurred_at": "2026-02-24T10:30:00Z",
    "created_at": "2026-02-24T10:30:01Z"
  }
  ```
- `401 Unauthorized`
- `404 Not Found`

#### **GET /health**
Health check endpoint (no auth required).

**Responses**:
- `200 OK` — Service healthy, includes version and DB pool stats
- `503 Service Unavailable` — Database unreachable

#### **GET /metrics**
Prometheus metrics endpoint (no auth required).

### Admin Endpoints

#### **POST /admin/api-keys**
Create a new API key with client metadata.

**Headers**:
- `X-API-Key: <admin-key>` (required, scope: `admin`)

**Request Body**:
```json
{
  "name": "Acme Corp - Production",
  "scopes": ["events:write", "events:read"],
  "metadata": {
    "client_id": "acme-corp",
    "organization": "Acme Corporation",
    "contact_email": "api@acme.com"
  }
}
```

**Response (201 Created)**:
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "name": "Acme Corp - Production",
  "scopes": ["events:write", "events:read"],
  "metadata": {
    "client_id": "acme-corp",
    "organization": "Acme Corporation",
    "contact_email": "api@acme.com"
  },
  "key": "abc123...xyz"
}
```

⚠️ **Important**: The raw `key` is returned only once on creation and cannot be retrieved later.

#### **GET /admin/api-keys**
List all API keys (without raw key values).

**Headers**:
- `X-API-Key: <admin-key>` (required, scope: `admin`)

#### **DELETE /admin/api-keys/:id**
Revoke an API key (sets `is_active` to false).

**Headers**:
- `X-API-Key: <admin-key>` (required, scope: `admin`)

### Supported Event Types

| Event Type | Payload Schema | Channel | Behavior |
|------------|----------------|---------|----------|
| `UserRegistered` | `{user_id, email, name}` | Email | 1 email per event |
| `PasswordResetRequested` | `{user_id, email}` | Email | 1 email per event |
| `OrderPaid` | `{order_id, user_id, amount, currency}` | Email | 1 email per event |
| `OrderShipped` | `{order_id, user_id, carrier, tracking_number}` | Email | 1 email per event |
| `InvoiceIssued` | `{condominium_id, condominium_name, invoice_month, due_date, currency, recipients[]}` | Email | Fan-out: 1 event → N emails (max 500 recipients) |
| `InvoiceSummary` | `{condominium_id, condominium_name, invoice_month, total_units, total_amount, currency, whatsapp_group_id, message}` | WhatsApp | 1 group message per event |

#### Example: Submit an invoice event (fan-out)

```bash
curl -X POST http://localhost:8080/api/v1/events \
  -H "X-API-Key: your-api-key-here" \
  -H "Idempotency-Key: $(uuidgen)" \
  -H "Content-Type: application/json" \
  -d '{
    "event_type": "InvoiceIssued",
    "payload": {
      "condominium_id": "condo-001",
      "condominium_name": "Residencias Sol",
      "invoice_month": "2026-03",
      "due_date": "2026-04-10",
      "currency": "USD",
      "recipients": [
        {"email": "maria@email.com", "name": "Maria Garcia", "unit_code": "1-A", "amount": 150.00},
        {"email": "jose@email.com", "name": "Jose Lopez", "unit_code": "2-B", "amount": 200.00}
      ]
    }
  }'
```

This creates one email notification per recipient. The worker delivers them asynchronously.

#### Example: Submit a WhatsApp group summary

```bash
curl -X POST http://localhost:8080/api/v1/events \
  -H "X-API-Key: your-api-key-here" \
  -H "Idempotency-Key: $(uuidgen)" \
  -H "Content-Type: application/json" \
  -d '{
    "event_type": "InvoiceSummary",
    "payload": {
      "condominium_id": "condo-001",
      "condominium_name": "Residencias Sol",
      "invoice_month": "2026-03",
      "total_units": 180,
      "total_amount": 27000.00,
      "currency": "USD",
      "whatsapp_group_id": "group-xyz",
      "message": "Se cargo el recibo de marzo 2026. Total: $27,000 (180 unidades)"
    }
  }'
```

Full API documentation: [docs/api.apib](docs/api.apib)

---

## 🔧 Configuration

All configuration is via **environment variables**. See [.env.example](.env.example) for all available options.

### Required Variables

| Variable | Description | Example |
|----------|-------------|---------|
| `PG_DSN` | PostgreSQL connection string | `postgres://user:pass@localhost:5432/events?sslmode=disable` |

**Note**: API keys are managed via the `/admin/api-keys` endpoint (requires an initial admin key). See [API Key Management](#api-key-management) section.

### API Server

| Variable | Default | Description |
|----------|---------|-------------|
| `API_ADDR` | `:8080` | HTTP listen address |
| `APP_ENV` | `development` | Environment (`production` enforces stricter validation) |
| `LOG_LEVEL` | `info` | Log level (`debug`, `info`, `warn`, `error`) |
| `MAX_BODY_BYTES` | `1048576` | Max request body size (1MB) |
| `READ_TIMEOUT` | `15` | HTTP read timeout (seconds) |
| `WRITE_TIMEOUT` | `15` | HTTP write timeout (seconds) |
| `SHUTDOWN_TIMEOUT` | `30` | Graceful shutdown timeout (seconds) |
| `TRUSTED_PROXIES` | — | Comma-separated trusted proxy IPs (e.g., `10.0.0.0/8`) |

### API Key Management

API keys are created and managed via admin endpoints. Each key includes:
- **Scopes**: `events:write`, `events:read`, `admin`
- **Metadata**: Client tracking information (`client_id`, `organization`, `contact_email`)

See [Admin Endpoints](#admin-endpoints) for details on creating API keys.

### TLS

| Variable | Default | Description |
|----------|---------|-------------|
| `TLS_CERT_FILE` | — | Path to TLS certificate (both required together) |
| `TLS_KEY_FILE` | — | Path to TLS private key |

### CORS

| Variable | Default | Description |
|----------|---------|-------------|
| `CORS_ALLOW_ALL` | `true` (dev), `false` (prod) | Allow all origins |
| `CORS_ALLOWED_ORIGINS` | — | Comma-separated allowed origins (when `CORS_ALLOW_ALL=false`) |
| `CORS_ALLOWED_HEADERS` | `Origin,Content-Length,...` | Allowed headers |

### Database

| Variable | Default | Description |
|----------|---------|-------------|
| `DB_QUERY_TIMEOUT` | `5` | Per-query timeout (seconds) |
| `DB_POOL_MAX_CONNS` | `10` | Max connections |
| `DB_POOL_MIN_CONNS` | `2` | Min idle connections |
| `DB_POOL_MAX_CONN_LIFETIME` | `3600` | Max connection reuse time (seconds) |
| `DB_POOL_MAX_CONN_IDLE_TIME` | `1800` | Max connection idle time (seconds) |

### Worker

| Variable | Default | Description |
|----------|---------|-------------|
| `WORKER_PROCESS_INTERVAL` | `5` | Event processing interval (seconds) |
| `WORKER_DELIVER_INTERVAL` | `3` | Notification delivery interval (seconds) |
| `WORKER_BATCH_SIZE` | `50` | Max events/notifications per batch |
| `WORKER_MAX_RETRIES` | `5` | Max delivery attempts |

### SMTP (Worker Only)

| Variable | Default | Description |
|----------|---------|-------------|
| `SMTP_HOST` | — | SMTP server hostname |
| `SMTP_PORT` | `587` | SMTP port (typically 587 for TLS) |
| `SMTP_USER` | — | SMTP username |
| `SMTP_PASSWORD` | — | SMTP password |
| `SMTP_FROM` | — | From address for outgoing emails |

### WhatsApp (Worker Only)

| Variable | Default | Description |
|----------|---------|-------------|
| `WHATSAPP_API_URL` | — | Base URL of the WhatsApp messaging API |
| `WHATSAPP_API_TOKEN` | — | Bearer token for WhatsApp API authentication |

When `WHATSAPP_API_URL` is not set, the worker starts without WhatsApp support. Any pending WhatsApp notifications will fail with a descriptive error and follow the normal retry/failure cycle.

### OpenTelemetry (Tracing)

| Variable | Default | Description |
|----------|---------|-------------|
| `OTEL_SERVICE_NAME` | `event-notification-service` | Service name in traces |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | — | OTLP HTTP endpoint (e.g. `localhost:4318`). Empty = stdout in dev, no-op in prod |

**Behavior by environment**:
- **Development** (`APP_ENV=development`): Traces exported to stdout as pretty-printed JSON
- **Production** with `OTEL_EXPORTER_OTLP_ENDPOINT` set: Traces exported to OTLP collector (Jaeger, Tempo, etc.)
- **Production** without `OTEL_EXPORTER_OTLP_ENDPOINT`: Tracing is no-op (minimal overhead)

---

## 🧪 Testing

### Unit Tests

Run all unit tests (no database required):

```bash
make test
# Or: go test -race -cover ./internal/...
```

**Coverage**: **≥ 80%** (enforced by CI)

Core business logic (domain, application, infrastructure) has comprehensive test coverage. The `cmd/` packages (composition roots) are excluded from coverage calculations as they primarily wire dependencies.

### Integration Tests

Integration tests require a live PostgreSQL instance:

```bash
# Start Postgres
docker-compose up -d postgres

# Run integration tests
make test-integration
# Or: go test -tags=integration -race ./internal/tests/
```

**Integration test coverage**:
- Repository layer: create, idempotency at DB level
- Use case + repository: all event types, payload validation, retry logic
- Worker end-to-end: event submission → processing → email delivery

### Linting

```bash
make lint
# Or: golangci-lint run
```

Configuration: [.golangci.yml](.golangci.yml)

### Race Detector

```bash
make test-race
# Or: go test -race ./internal/...
```

Useful when validating middleware, async updates, and request-scoped concurrency.

### Benchmarks

```bash
make bench
make bench-validate-event
```

The first command runs all benchmarks under `internal/`. The second focuses on the event validation hot path and reports allocations with `-benchmem`.

### pprof

Generate CPU and memory profiles for the validation benchmark:

```bash
make pprof-validate-event-cpu
make pprof-validate-event-mem
```

Inspect the generated profiles:

```bash
go tool pprof -http=:0 ./internal/application/validation.test profiles/validate_event_cpu.prof
go tool pprof -http=:0 ./internal/application/validation.test profiles/validate_event_mem.prof
```

Profiles are written to `profiles/` and are useful for confirming allocation changes before and after optimization work.

---

## 🏭 Deployment

### Docker Build

The project includes a multi-stage Dockerfile with separate targets:

```bash
# Build API image
docker build --target api -t event-service-api:latest .

# Build worker image
docker build --target worker -t event-service-worker:latest .
```

### Kubernetes

Kubernetes manifests are provided in `deploy/k8s/`:

- `deployment-api.yaml` — API server deployment (3 replicas)
- `deployment-worker.yaml` — Worker deployment (2 replicas)
- `service.yaml` — LoadBalancer service for API
- `configmap.yaml` — Non-sensitive configuration
- `secret.yaml` — Sensitive credentials (template with placeholders)

**Deploy**:

```bash
# Update secrets with real values
kubectl apply -f deploy/k8s/secret.yaml

# Apply all manifests
kubectl apply -f deploy/k8s/

# Check rollout status
kubectl rollout status deployment/event-service-api
kubectl rollout status deployment/event-service-worker
```

### CI/CD

GitHub Actions workflows:

#### `.github/workflows/ci.yml` (on every push/PR)
1. **Lint**: `golangci-lint` (v2 config)
2. **Test**: Unit tests with race detector, **coverage ≥ 80%** (enforced threshold)
3. **Build**: Compile both `bin/event-service` (API) and `bin/worker`

#### `.github/workflows/deploy.yml` (on push to main)
1. **Migrate**: Apply DB migrations (requires `PG_DSN` secret)
2. **Build & Push**: Build Docker images → push to GitHub Container Registry
3. **Deploy**: `kubectl apply` (requires `KUBECONFIG` secret)

**Required GitHub Secrets**:
- `PG_DSN` — Production database connection string
- `KUBECONFIG` — Kubernetes cluster credentials

---

## 🛠️ Development

### Makefile Targets

```bash
make build                # Build API binary (version injection)
make build-worker         # Build worker binary
make test                 # Run unit tests
make test-race            # Run unit tests with the race detector
make bench                # Run all internal benchmarks with allocation stats
make bench-validate-event # Benchmark the validation hot path only
make pprof-validate-event-cpu # Generate CPU profile for ValidateEvent benchmark
make pprof-validate-event-mem # Generate memory profile for ValidateEvent benchmark
make test-integration     # Run integration tests (requires Postgres)
make lint                 # Run golangci-lint
make migrate              # Apply DB migrations
make migrate-down         # Rollback last migration
make docs                 # Render API Blueprint to HTML (requires aglio)
make clean                # Remove build artifacts
```

### Database Migrations

Migrations are in `internal/infrastructure/postgres/migrations/`:

```
001_create_events.up.sql / .down.sql
002_create_notifications.up.sql / .down.sql
003_add_event_status.up.sql / .down.sql
004_create_api_keys.up.sql / .down.sql
005_add_api_keys_metadata.up.sql / .down.sql    # API key client tracking
006_add_events_client_id.up.sql / .down.sql      # Event client tracking
007_add_whatsapp_channel.up.sql / .down.sql       # WhatsApp notification channel
```

**Apply all pending migrations**:
```bash
make migrate
```

**Rollback last migration**:
```bash
make migrate-down
```

**Create new migration**:
```bash
# Manually create files: {version}_{name}.up.sql and {version}_{name}.down.sql
touch internal/infrastructure/postgres/migrations/004_add_index.up.sql
touch internal/infrastructure/postgres/migrations/004_add_index.down.sql
```

---

## 📊 Monitoring & Observability

### Observability Stack

The project includes a full observability stack via `docker-compose`:

```bash
docker-compose up -d
```

| Service | Port | Function |
|---|---|---|
| **app** | `:8080` | API server |
| **worker** | `:9090` | Async worker (metrics on /metrics) |
| **postgres** | `:5432` | Database |
| **prometheus** | `:9091` | Scrapes metrics every 15s, stores in TSDB |
| **tempo** | `:3200` / `:4318` | Receives traces via OTLP, stores them |
| **loki** | `:3100` | Log aggregation |
| **promtail** | — | Collects container logs, ships to Loki |
| **grafana** | `:3000` | Dashboard for metrics, traces, and logs (admin/admin) |

Data flow:

```
app/worker ──metrics──→ Prometheus (:9091)  ──→ Grafana (:3000)
app/worker ──OTLP────→ Tempo (:4318)       ──→ Grafana
app/worker ──stdout──→ Promtail → Loki     ──→ Grafana
```

Configuration files are in `observability/`:
- `prometheus.yml` — scrape targets
- `tempo.yml` — OTLP receiver and storage
- `loki.yml` — log storage
- `promtail.yml` — Docker log collection
- `grafana/provisioning/datasources/` — pre-configured datasources (Prometheus, Tempo, Loki)

### OpenTelemetry Tracing

Distributed tracing is implemented using **OpenTelemetry SDK** with OTLP/HTTP export:

| Env Var | Default | Description |
|---|---|---|
| `OTEL_SERVICE_NAME` | `event-notification-service` | Service name in traces |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | — | OTLP HTTP endpoint (e.g. `localhost:4318`). Empty = stdout (dev) or no-op (prod) |

**What's instrumented**:
- ✅ **HTTP requests** — Automatic spans via `otelgin` middleware (method, path, status, latency)
- ✅ **Use cases** — `SubmitEvent`, `ProcessEvents`, `DeliverNotifications` with business attributes (event_type, notification_id, etc.)
- ✅ **Infrastructure** — Database operations (Create, GetByID), SMTP sender, WhatsApp sender
- ✅ **Context propagation** — TraceContext + Baggage via W3C standard headers

**Viewing traces**:
- **Development**: Traces printed to stdout as JSON (when `APP_ENV=development`)
- **Production**: Export to Grafana Tempo via OTLP (see `docker-compose.yml`)
- **Correlation**: Each trace includes `request_id` for log correlation

### Prometheus Integration

Both API and worker expose Prometheus metrics:

**Scrape API metrics**:
```yaml
# prometheus.yml
scrape_configs:
  - job_name: 'event-service-api'
    static_configs:
      - targets: ['localhost:8080']
    metrics_path: '/metrics'
```

**Scrape worker metrics**:
```yaml
  - job_name: 'event-service-worker'
    static_configs:
      - targets: ['localhost:9090']
    metrics_path: '/metrics'
```

### Key Metrics to Monitor

**API Health**:
- `http_requests_total{status="5xx"}` — Server errors
- `http_request_duration_seconds{quantile="0.99"}` — P99 latency
- `http_errors_total{code="429"}` — Rate limit hits
- `events_submitted_total{result="error"}` — Event submission failures

**Worker Health**:
- `events_processed_total{result="error"}` — Processing failures
- `notifications_delivered_total{channel="email",result="error"}` — Delivery failures
- `worker_cycles_total{result="error"}` — Worker loop errors
- `postgres_pool_acquire_duration_seconds` — DB connection latency

**Database Health**:
- `postgres_pool_total_conns` vs `postgres_pool_max_conns` — Connection saturation
- `postgres_pool_idle_conns` — Connection pool efficiency

### Structured Logging

All logs are **JSON-formatted** (Zap) with consistent fields. Logs and traces are **correlated** via `request_id` and OpenTelemetry trace IDs:

```json
{
  "level": "info",
  "ts": "2026-03-31T10:30:00.123Z",
  "caller": "http/handler.go:45",
  "msg": "event submitted",
  "request_id": "550e8400-e29b-41d4-a716-446655440000",
  "trace_id": "abc123def456...",
  "span_id": "789xyz...",
  "event_id": "123e4567-e89b-12d3-a456-426614174000",
  "event_type": "UserRegistered",
  "client_id": "acme-corp"
}
```

**Audit logs** (API Key middleware):
```json
{
  "level": "info",
  "ts": "2026-03-31T10:30:00.123Z",
  "msg": "auth success",
  "event": "auth",
  "api_key_name": "Acme Corp - Production",
  "client_id": "acme-corp",
  "remote_ip": "203.0.113.42",
  "request_id": "550e8400-e29b-41d4-a716-446655440000"
}
```

**Log aggregation**: Logs are collected by Promtail and shipped to Loki (see `docker-compose.yml`). In Grafana, you can jump from a log entry to its corresponding trace using the trace_id field.

---

## 🔐 Security Considerations

### API Key Authentication

- API keys are stored as **SHA-256 hashes** (never plaintext)
- **Scope-based authorization**: `events:write`, `events:read`, `admin`
- **Client tracking**: Each key has metadata with `client_id` for auditing
- Keys can be **revoked** at any time via `DELETE /admin/api-keys/:id`
- **Audit logging**: All authentication attempts logged with IP and client_id

### Rate Limiting

- **Infrastructure-level**: Rate limiting should be configured at your load balancer or API gateway (e.g., NGINX, AWS ALB, CloudFlare)
- Application-level rate limiting was removed in commit `318e50f` to reduce complexity and align with modern best practices
- The service still returns `429 Too Many Requests` responses when upstream rate limits are hit

### Security Headers

Automatically applied to all API responses:
- `X-Content-Type-Options: nosniff`
- `X-Frame-Options: DENY`
- `Strict-Transport-Security` (when HTTPS is enabled)
- `Cache-Control: no-store` (on authenticated routes)

### Production Checklist

Before deploying to production:

- [ ] Set `APP_ENV=production`
- [ ] Create an initial **admin API key** for managing other keys
- [ ] Configure `TRUSTED_PROXIES` to match your load balancer
- [ ] Set `CORS_ALLOW_ALL=false` and specify `CORS_ALLOWED_ORIGINS`
- [ ] Enable TLS via `TLS_CERT_FILE` and `TLS_KEY_FILE` (or terminate at load balancer)
- [ ] Set `HSTS_ENABLED=true` (when using HTTPS)
- [ ] Review resource limits in Kubernetes manifests
- [ ] Set up Prometheus scraping and alerting
- [ ] Configure SMTP credentials for email delivery
- [ ] Configure WhatsApp API credentials (`WHATSAPP_API_URL`, `WHATSAPP_API_TOKEN`) if using WhatsApp notifications
- [ ] Set up OpenTelemetry collector endpoint (`OTEL_EXPORTER_OTLP_ENDPOINT`) for distributed tracing
- [ ] Apply database migrations (up to `007_add_whatsapp_channel`)
- [ ] Create API keys for all clients with proper metadata
- [ ] Configure infrastructure-level rate limiting at load balancer/API gateway
- [ ] Test graceful shutdown behavior under load

---

## 📚 Additional Documentation

- **[CLAUDE.md](CLAUDE.md)** — Architecture deep dive, coding conventions, testing patterns
- **[AGENT.md](AGENT.md)** — Project state tracker (all 9 phases completed)
- **[docs/api.apib](docs/api.apib)** — Full API specification in API Blueprint format

### Architecture Highlights

**Validation Responsibilities**:

| Layer | What it validates |
|-------|-------------------|
| **Domain** (`entities.NewEvent`) | Invariants: type not empty, type in `ValidEventTypes()`, idempotency key not empty |
| **Application** (`validation.ValidateEvent`) | Payload shape: required fields per event type, format rules (email contains `@`), amount > 0, recipients array (max 500) |

**Error Handling**:
- `internal/pkg/apperror` defines error codes (`invalid_argument`, `conflict`, `not_found`, `timeout`, `canceled`, `rate_limited`, etc.)
- `internal/infrastructure/http/errmap` maps error codes to HTTP status codes
- Handler never sets HTTP status directly from business logic
- Context errors (`context.Canceled` → 499, `context.DeadlineExceeded` → 504) are preserved through the stack

**Idempotency**:
- Enforced via `UNIQUE(idempotency_key, type)` DB constraint
- INSERT uses `ON CONFLICT DO UPDATE SET id = events.id RETURNING id` to return original ID on duplicates
- Same idempotency key with different event type is allowed (different events)

---

## 🤝 Contributing

This project is feature-complete and in maintenance mode. All 9 phases outlined in [AGENT.md](AGENT.md) are implemented:

- ✅ Phase 1-2: Data model + domain/ports
- ✅ Phase 3: Use cases (ProcessEvents, DeliverNotifications)
- ✅ Phase 4: Email infrastructure (SMTP adapter, templates)
- ✅ Phase 5: Async worker
- ✅ Phase 6: Production hardening (rate limiting, graceful shutdown, config validation)
- ✅ Phase 7: Observability (Prometheus metrics, audit logging)
- ✅ Phase 8: Testing (34.4% coverage, integration tests)
- ✅ Phase 9: Deployment (Kubernetes manifests, CI/CD pipeline)

If you find bugs or have suggestions, please open an issue.

---

## 📝 License

This project is licensed under the MIT License. See [LICENSE](LICENSE) for details.

---

## 🙏 Acknowledgments

- Built with [Go](https://go.dev/)
- Web framework: [Gin](https://github.com/gin-gonic/gin)
- Database: [PostgreSQL](https://www.postgresql.org/) via [pgx](https://github.com/jackc/pgx)
- Metrics: [Prometheus](https://prometheus.io/) via [client_golang](https://github.com/prometheus/client_golang)
- Logging: [Zap](https://github.com/uber-go/zap)
- JWT: [golang-jwt/jwt](https://github.com/golang-jwt/jwt)
- Inspired by hexagonal architecture and domain-driven design patterns

---

**Project Status**: ✅ **Production Ready** — All phases complete, ready for deployment.

For questions or support, please open an issue on GitHub.
