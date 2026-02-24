# Event Notification Service

A production-ready, high-performance event notification service built in Go with hexagonal architecture. Receives domain events via REST API, persists them in PostgreSQL, and processes them asynchronously through a worker to deliver email notifications.

[![CI](https://github.com/rachelJG/event-notification-service/workflows/CI/badge.svg)](https://github.com/rachelJG/event-notification-service/actions)
[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go)](https://go.dev/)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

---

## 🎯 Overview

This service implements a **complete event-driven notification pipeline** with:

- **REST API** for event ingestion with idempotency guarantees
- **Asynchronous Worker** for event processing and notification delivery
- **Email notifications** with templating support per event type
- **Production hardening**: rate limiting, JWT auth, CORS, security headers, graceful shutdown
- **Full observability**: Prometheus metrics, structured logging, health checks
- **Clean architecture**: Hexagonal/Ports & Adapters pattern with zero domain dependencies

### Key Design Decisions

| Aspect | Choice | Rationale |
|--------|--------|-----------|
| **API Style** | REST (HTTP/JSON) | Simple, wide tooling support, sufficient for event submission |
| **Storage** | PostgreSQL 15 | ACID guarantees, transactional workflows, idempotency via constraints |
| **Processing** | Async worker (polling) | Decouples producers from delivery, resilient to external failures |
| **Architecture** | Hexagonal | Domain isolation, testability, infrastructure independence |
| **Auth** | JWT (HS256) | Stateless, industry standard, configurable validation |

---

## 🏗️ Architecture

### System Components

```
┌──────────────┐                 ┌──────────────┐
│   Producer   │                 │   Consumer   │
│  (External)  │                 │  (External)  │
└──────┬───────┘                 └──────▲───────┘
       │                                │
       │ HTTP POST                      │ Email/Webhook
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
   │  email/              │ │                      │ │                    │
   │   ├─ smtp_adapter.go │ │  validation/         │ │                    │
   │   └─ templates.go    │ │   └─ validate_       │ │                    │
   │                      │ │      event.go        │ │                    │
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
│   │   └── logger/                    # Zap logger factory
│   ├── pkg/
│   │   └── apperror/                  # Typed error taxonomy
│   └── tests/                         # Integration tests (//go:build integration)
├── deploy/
│   └── k8s/                           # Kubernetes manifests
├── docs/
│   └── api.apib                       # API Blueprint documentation
├── .github/workflows/
│   ├── ci.yml                         # Lint, test, coverage (34%)
│   └── deploy.yml                     # Migrations → Build → Deploy pipeline
├── docker-compose.yml                 # Local development setup
├── Makefile                           # Build, test, lint, migrate targets
├── CLAUDE.md                          # Codebase conventions for AI agents
└── AGENT.md                           # Project state tracker (all phases DONE)
```

---

## ✨ Features

### Core Functionality

- ✅ **Event Ingestion**: REST API with JWT authentication and idempotency keys
- ✅ **Supported Event Types**:
  - `user.signup` — User registration notifications
  - `password.reset` — Password reset emails
  - `payment.received` — Payment confirmation
  - `order.shipped` — Shipping notifications
- ✅ **Idempotency**: Guaranteed via `UNIQUE(idempotency_key, type)` DB constraint
- ✅ **Async Processing**: Worker polls pending events every 5 seconds
- ✅ **Email Delivery**: SMTP adapter with per-event-type templates
- ✅ **Retry Logic**: Configurable max retries with exponential backoff

### Production Hardening

- 🔒 **Security**:
  - JWT authentication (HS256, min 32-byte secret)
  - Optional issuer/audience validation
  - Rate limiting with `X-Forwarded-For` support
  - Security headers (HSTS, X-Content-Type-Options, X-Frame-Options)
  - Trusted proxy configuration
- 🛡️ **Reliability**:
  - Graceful shutdown (configurable timeout)
  - Context-aware timeouts (DB queries, HTTP requests)
  - Circuit breaker patterns ready
  - Dead letter queue support (status: failed)
- 📊 **Observability**:
  - Prometheus metrics (HTTP, DB pool, business metrics)
  - Structured logging (Zap) with request IDs
  - JWT audit logging (auth success/failure with IP)
  - Health check endpoints (`/health` with DB stats)

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

### Endpoints

#### **POST /api/v1/events**
Submit a new event for processing.

**Headers**:
- `Authorization: Bearer <jwt>` (required)
- `Idempotency-Key: <uuid>` (required)
- `Content-Type: application/json` (required)

**Request Body**:
```json
{
  "event_type": "user.signup",
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
- `401 Unauthorized` — Missing or invalid JWT
- `409 Conflict` — Duplicate event (idempotent: returns original ID)
- `429 Too Many Requests` — Rate limit exceeded (includes `Retry-After` header)

**Example**:
```bash
curl -X POST http://localhost:8080/api/v1/events \
  -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..." \
  -H "Idempotency-Key: $(uuidgen)" \
  -H "Content-Type: application/json" \
  -d '{
    "event_type": "user.signup",
    "payload": {
      "user_id": "12345",
      "email": "user@example.com",
      "name": "John Doe"
    }
  }'
```

#### **GET /api/v1/events/:id**
Retrieve a previously submitted event.

**Headers**:
- `Authorization: Bearer <jwt>` (required)

**Responses**:
- `200 OK`
  ```json
  {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "type": "user.signup",
    "payload": {"user_id": "12345", "email": "user@example.com", "name": "John Doe"},
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

### Supported Event Types

| Event Type | Payload Schema | Email Template |
|------------|----------------|----------------|
| `user.signup` | `{user_id, email, name}` | Welcome email |
| `password.reset` | `{email, reset_link}` | Password reset instructions |
| `payment.received` | `{email, amount, currency}` | Payment confirmation |
| `order.shipped` | `{email, order_id, tracking_number}` | Shipping notification |

Full API documentation: [docs/api.apib](docs/api.apib)

---

## 🔧 Configuration

All configuration is via **environment variables**. See [.env.example](.env.example) for all available options.

### Required Variables

| Variable | Description | Example |
|----------|-------------|---------|
| `PG_DSN` | PostgreSQL connection string | `postgres://user:pass@localhost:5432/events?sslmode=disable` |
| `JWT_SECRET` | JWT signing secret (min 32 bytes) | `your-secret-key-at-least-32-bytes-long` |

### API Server

| Variable | Default | Description |
|----------|---------|-------------|
| `API_ADDR` | `:8080` | HTTP listen address |
| `APP_ENV` | `development` | Environment (`production` enforces stricter validation) |
| `LOG_LEVEL` | `info` | Log level (`debug`, `info`, `warn`, `error`) |
| `MAX_BODY_BYTES` | `1048576` | Max request body size (1MB) |
| `RATE_LIMIT_RPS` | `10` | Requests per second per IP |
| `RATE_LIMIT_BURST` | `20` | Burst allowance |
| `READ_TIMEOUT` | `15` | HTTP read timeout (seconds) |
| `WRITE_TIMEOUT` | `15` | HTTP write timeout (seconds) |
| `SHUTDOWN_TIMEOUT` | `30` | Graceful shutdown timeout (seconds) |
| `TRUSTED_PROXIES` | — | Comma-separated trusted proxy IPs (e.g., `10.0.0.0/8`) |

### JWT Authentication

| Variable | Default | Description |
|----------|---------|-------------|
| `JWT_SECRET` | — | HS256 signing secret (**required** in production) |
| `JWT_ISSUER` | — | Expected `iss` claim (optional) |
| `JWT_AUDIENCE` | — | Expected `aud` claim (optional) |

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

---

## 🧪 Testing

### Unit Tests

Run all unit tests (no database required):

```bash
make test
# Or: go test -race -cover ./...
```

**Coverage**: 34.4% (threshold: 34%)

Coverage is conservative because `cmd/`, `config/`, and `logger/` packages (composition roots and constructors) have no unit tests. Core business logic has comprehensive coverage.

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
1. **Lint**: `golangci-lint`
2. **Test**: Unit tests with race detector, coverage ≥ 34%
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
001_create_events_table.up.sql
001_create_events_table.down.sql
002_create_notifications_table.up.sql
002_create_notifications_table.down.sql
003_add_status_to_events.up.sql
003_add_status_to_events.down.sql
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

All logs are JSON-formatted (Zap) with consistent fields:

```json
{
  "level": "info",
  "ts": "2026-02-24T10:30:00.123Z",
  "caller": "http/handler.go:45",
  "msg": "event submitted",
  "request_id": "550e8400-e29b-41d4-a716-446655440000",
  "event_id": "123e4567-e89b-12d3-a456-426614174000",
  "event_type": "user.signup",
  "user_id": "12345"
}
```

**Audit logs** (JWT middleware):
```json
{
  "level": "info",
  "ts": "2026-02-24T10:30:00.123Z",
  "msg": "auth success",
  "event": "auth",
  "subject": "user-123",
  "ip": "203.0.113.42"
}
```

---

## 🔐 Security Considerations

### Authentication

- JWT tokens must use **HS256** algorithm (RS256, ES256, `none` are rejected)
- `exp` claim is **required** (tokens without expiry are rejected)
- Minimum secret length: **32 bytes** (enforced in config validation)
- Optional issuer/audience validation via `JWT_ISSUER` and `JWT_AUDIENCE`

### Rate Limiting

- Per-IP rate limiting with `X-Forwarded-For` support
- Configurable via `RATE_LIMIT_RPS` and `RATE_LIMIT_BURST`
- Returns `429 Too Many Requests` with `Retry-After` header

### Security Headers

Automatically applied to all API responses:
- `X-Content-Type-Options: nosniff`
- `X-Frame-Options: DENY`
- `Strict-Transport-Security` (when HTTPS is enabled)
- `Cache-Control: no-store` (on authenticated routes)

### Production Checklist

Before deploying to production:

- [ ] Set `APP_ENV=production`
- [ ] Use a strong `JWT_SECRET` (min 32 bytes, randomly generated)
- [ ] Configure `TRUSTED_PROXIES` to match your load balancer
- [ ] Set `CORS_ALLOW_ALL=false` and specify `CORS_ALLOWED_ORIGINS`
- [ ] Enable TLS via `TLS_CERT_FILE` and `TLS_KEY_FILE` (or terminate at load balancer)
- [ ] Set `HSTS_ENABLED=true` (when using HTTPS)
- [ ] Configure `JWT_ISSUER` and `JWT_AUDIENCE` for additional validation
- [ ] Review resource limits in Kubernetes manifests
- [ ] Set up Prometheus scraping and alerting
- [ ] Configure SMTP credentials for email delivery
- [ ] Apply database migrations before deploying code
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
| **Application** (`validation.ValidateEvent`) | Payload shape: required fields per event type, format rules (email contains `@`), amount > 0 |

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
