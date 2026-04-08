# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build & Development Commands

```bash
make build               # Compile binary with git-derived VERSION and COMMIT ldflags
make test                # Run unit tests (no DB required)
make test-integration    # Run integration tests (needs Postgres, loads .env)
make lint                # Run golangci-lint
make migrate             # Apply DB migrations (loads .env)
make migrate-down        # Roll back last migration (loads .env)
make docs                # Generate Swagger API documentation
go run ./cmd/api         # Start the API server
docker-compose up -d     # Start app + Postgres locally
```

Run a single test:
```bash
go test ./internal/application/validation/ -run TestValidateEvent
```

Integration tests use the `//go:build integration` tag and require a live Postgres instance (`PG_DSN` env var).

## Architecture

Hexagonal Architecture (Domain / Application / Infrastructure). The domain layer has zero external dependencies.

```
cmd/api/main.go                    → Entry point, composition root, signal handling, graceful shutdown
internal/config/                   → Env-var-based configuration (shared, layer-independent)
internal/domain/entities/          → Event entity, EventType value object, payload types, NewEvent constructor
internal/domain/ports/             → Output port interfaces (EventRepository: Create, GetByID)
internal/application/ports/        → Input port interfaces (EventService)
internal/pkg/apperror/             → Typed error taxonomy (AppError with codes: invalid_argument, conflict, not_found, timeout, canceled, rate_limited, etc.)
internal/application/usecases/     → Business logic (EventService implementation with SubmitEvent, GetEvent methods)
internal/application/validation/   → Payload shape validation per event type (application concern, not domain)
internal/infrastructure/http/      → Gin router, handler, DTOs, RouterOptions, HealthChecker interface, middleware stack, error-to-HTTP mapping, business metrics
internal/infrastructure/postgres/  → EventRepository implementation using pgxpool (Create, GetByID, mapDBError)
internal/infrastructure/whatsapp/  → WhatsApp sender implementation (HTTP-based API client)
internal/infrastructure/logger/    → Zap logger factory
```

**Dependency flow:** `cmd/api (composition root) → infrastructure → ports ← usecases ← entities`. Infrastructure depends on ports; domain and application never import infrastructure. The HTTP adapter uses `RouterOptions` instead of importing the `config` package directly, and the health check endpoint depends on a `HealthChecker` port interface rather than `*pgxpool.Pool`.

### Port placement

- **Output ports** (what the application needs from infrastructure) live in `domain/ports/`. The domain defines the contract; infrastructure implements it.
- **Input ports** (what the application exposes to driving adapters) live in `application/ports/`.
- **Operational interfaces** used only by a single adapter (e.g. `HealthChecker`) are defined at the point of use, inside that adapter's package.

### Validation responsibilities

Validation is split across two layers with a clear boundary:

| Layer | Where | What it validates |
|---|---|---|
| Domain | `entities.NewEvent` | Invariants: type not empty, type is in `ValidEventTypes()`, idempotency key not empty |
| Application | `validation.ValidateEvent` | Payload shape: required fields per event type, format rules (e.g. email contains `@`), amount > 0, recipients array (max 500) |

The domain does **not** validate payload content — that would require importing `encoding/json`, breaking the zero-dependency rule. Payload bytes are opaque to the domain.

The use case calls `ValidateEvent` **before** `NewEvent`: validate input first, construct the domain object only when input is known to be valid.

## Key Patterns

- **Idempotency**: Enforced at DB level via `UNIQUE(idempotency_key, type)` index. INSERT uses `ON CONFLICT DO UPDATE SET id = events.id RETURNING id` to return the original ID on duplicates.
- **Error mapping**: `internal/pkg/apperror` defines error codes. `internal/infrastructure/http/errmap` maps them to HTTP status codes. Handler never sets HTTP status directly from business logic. `context.Canceled` maps to 499 (nginx client-closed-request convention); `context.DeadlineExceeded` maps to 504.
- **Middleware stack** (in order): Zap logger+recovery → request ID → security headers → Prometheus metrics → CORS → body limit → content-type enforcement → rate limiter. API Key auth is applied only to authenticated routes.
- **SQL safety**: `internal/infrastructure/postgres/sqlsafe.go` provides allowlist-based identifier sanitization for any dynamic SQL.
- **DB error mapping**: `mapDBError` in `postgres/event_repository.go` translates pgx/pgconn error codes to `AppError` (23505 unique_violation → conflict, 23502 not_null_violation → invalid_argument, pgx.ErrNoRows → not_found). Context errors pass through untouched so `errmap` can handle them at the HTTP layer.
- **Business metrics**: `events_submitted_total{event_type, result}` Prometheus counter in `infrastructure/http/metrics.go` tracks submitted events by type and outcome (success/error), separate from HTTP-level metrics in middleware.
- **Version injection**: `Version` and `Commit` package-level vars in `cmd/api/main.go` are populated at build time via `-ldflags`. `make build` derives them from `git describe` and `git rev-parse`. Exposed on `GET /health/ready`.
- **Fan-out notifications**: `InvoiceIssued` events fan out into N email notifications (one per recipient). `InvoiceSummary` events produce a single WhatsApp group notification. The worker's `ProcessEvents` use case handles dispatch by event type.
- **Multi-channel delivery**: `DeliverNotifications` dispatches by notification channel (`email` → SMTP, `whatsapp` → HTTP API). WhatsApp sender is optional — if not configured, WhatsApp notifications fail gracefully with a descriptive error.

## HTTP API

### `POST /api/v1/events`
Requires `X-API-Key: <api-key>` header (scope: `events:write`) and `Idempotency-Key: <uuid>` header.

The `client_id` from the API key's metadata is automatically captured and stored with the event for auditing and analytics.

- **202 Accepted** — event accepted; body `{"id": "..."}`, header `Location: /api/v1/events/{id}`
- **400 Bad Request** — invalid idempotency key, bad JSON, validation error, invalid UUID in path
- **401 Unauthorized** — missing/invalid API key or insufficient scopes
- **409 Conflict** — duplicate event (same idempotency key + type, idempotent: returns original id)
- **429 Too Many Requests** — rate limit exceeded

### `GET /api/v1/events/:id`
Requires `X-API-Key: <api-key>` header (scope: `events:read`).

- **200 OK** — body `{"id","type","payload","client_id","occurred_at","created_at"}`
  - `client_id` is included if the event was submitted with an authenticated API key (optional for backward compatibility)
- **400 Bad Request** — invalid UUID format
- **401 Unauthorized** — missing/invalid API key or insufficient scopes
- **404 Not Found** — event does not exist

### `GET /health/live`
No auth required. Liveness check — only verifies the process is running.

- **200 OK** — `{"status":"ok"}`

### `GET /health/ready`
No auth required. Readiness check — verifies the service can handle traffic (DB reachable).

- **200 OK** — `{"status":"ok","version":"...","commit":"...","db":{pool stats}}`
- **503 Service Unavailable** — database unreachable

### Admin endpoints

### `POST /admin/api-keys`
Requires API key with `admin` scope. Creates a new API key with client metadata.

**Request body:**
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

**Note:** `metadata.client_id` is required. Additional metadata fields are optional but recommended for tracking and auditing.

**Response (201 Created):**
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

**Important:** The raw `key` is returned only once on creation and cannot be retrieved later.

### `GET /admin/api-keys`
Requires API key with `admin` scope. Lists all API keys with their metadata (excluding the raw key).

### `DELETE /admin/api-keys/:id`
Requires API key with `admin` scope. Revokes an API key (sets `is_active` to false).

### `GET /metrics`
Prometheus metrics endpoint (no auth).

### `GET /swagger/index.html`
Swagger UI for interactive API documentation (no auth). After starting the API server, visit `http://localhost:8080/swagger/index.html` to explore and test all endpoints interactively.

## API Key Authentication

The primary authentication mechanism is **long-lived API Keys** for service-to-service communication. API keys are managed via the `/admin/api-keys` endpoints.

- Keys are stored as SHA-256 hashes in the `api_keys` table
- Each key has a set of **scopes** (e.g. `events:write`, `events:read`, `admin`)
- Each key has **metadata** (JSONB) containing client information:
  - `client_id` (required): unique identifier for the client/organization
  - `organization` (optional): human-readable organization name
  - `contact_email` (optional): contact email for the client
  - Additional custom fields as needed
- Middleware (`middleware.APIKeyAuth`) validates the key hash, checks required scopes, and propagates `client_id` to the request context for auditing
- The `client_id` is logged with each authenticated request and available via `c.GetString(middleware.ContextKeyClientID)`
- Keys can be revoked via `DELETE /admin/api-keys/:id`
- The API key repository is defined as a port interface in `domain/ports/`

### JWT Authentication (legacy)

JWT middleware (`middleware.JWTAuth`) is still present in the codebase but no longer used in the router. It was replaced by API Key auth for service-to-service communication.

## API Documentation

The API is documented using **Swagger (OpenAPI 3.0)** annotations in the code:

- Annotations are added to handler functions in `internal/infrastructure/http/handler.go` and `internal/infrastructure/http/admin_handler.go`
- DTO types in `internal/infrastructure/http/dto/` include field-level documentation
- Main API metadata (title, version, contact, license) is defined in `cmd/api/main.go`
- Run `make docs` to generate Swagger files (`docs/swagger.json`, `docs/swagger.yaml`, `docs/docs.go`)
- Swagger UI is available at `GET /swagger/index.html` when the server is running
- The generated `docs/docs.go` is imported in `server.go` to embed the documentation

To update the documentation after changing handlers or DTOs, run `make docs` and restart the server.

## TLS

Set `TLS_CERT_FILE` and `TLS_KEY_FILE` (both required together) to enable `ListenAndServeTLS`. When unset the server runs plain HTTP. TLS is typically terminated at the load balancer; these vars support direct TLS for environments that require it.

## Testing Approach

- Unit tests use fake/stub implementations (e.g., `fakeRepo` in handler and use case tests). No mocking library.
- HTTP handler tests use `httptest.NewRecorder` with a real Gin router.
- Prometheus counter assertions use `testutil.ToFloat64` from `prometheus/client_golang/prometheus/testutil`.
- Integration tests are in `internal/tests/` with the `integration` build tag.

### Unit test coverage

| Package | File | What it covers |
|---|---|---|
| `domain/entities` | `event_test.go` | `NewEvent` invariants: empty type, unsupported type, empty idempotency key, all valid types accepted |
| `application/validation` | `validate_event_test.go` | All 6 event types (valid + invalid fields + invalid JSON + invalid email + amount rules + recipients validation), empty type, unsupported type |
| `application/usecases` | `event_service_test.go` | EventService.SubmitEvent: success, payload error, repo error, missing idempotency key, empty type, unsupported type; EventService.GetEvent: success, empty id → invalid_argument, repo not_found error |
| `application/usecases` | `process_events_test.go` | ProcessEvents: single-recipient events, InvoiceIssued fan-out (N notifications), InvoiceSummary → WhatsApp notification |
| `application/usecases` | `deliver_notifications_test.go` | DeliverNotifications: email success, send error + retry, max retries → failed, WhatsApp success, WhatsApp error, WhatsApp not configured |
| `infrastructure/whatsapp` | `sender_test.go` | SendToGroup: success (verifies request format + auth header), API error response, context cancellation |
| `infrastructure/http` | `handler_test.go` | Missing/invalid idempotency key, success + Location header, service error, unauthorized, wrong content-type, rate limit, GET success, GET 404, GET unauthorized, metric increments |
| `infrastructure/http/errmap` | `errmap_test.go` | All AppError codes → HTTP status, context.DeadlineExceeded → 504, context.Canceled → 499 |
| `infrastructure/http/middleware` | `jwt_test.go` | HS256 success, subject in context, missing header, expired token, missing exp, RS256 rejected, issuer validation, audience validation, empty secret |
| `infrastructure/postgres` | `event_repository_test.go` | `mapDBError`: unique_violation → conflict, not_null_violation → invalid_argument, context errors pass-through, generic errors → internal |

### Integration test coverage (`internal/tests/`, requires Postgres)

| File | What it covers |
|---|---|
| `postgres_event_repository_integration_test.go` | Repository layer: create, idempotency at DB level |
| `submit_event_integration_test.go` | Use case + repository: all event types persisted, idempotency end-to-end, same key different type, invalid payload not persisted, unsupported type rejected, empty idempotency key rejected |

## Configuration

All config is via environment variables (see `.env.example`). Notable constraints enforced in `config.Validate()`:

| Rule | Detail |
|---|---|
| `JWT_SECRET` required | When `APP_ENV=production` |
| `JWT_SECRET` min length | 32 bytes when set |
| `TLS_CERT_FILE` / `TLS_KEY_FILE` | Must be set together or both empty |
| `CORS_ALLOW_ALL` | Cannot be combined with `CORS_ALLOWED_ORIGINS` |
| `HSTS_MAX_AGE_SECONDS` | Must be > 0 when HSTS is enabled |

Key env vars:

| Var | Default | Description |
|---|---|---|
| `JWT_SECRET` | — | HS256 signing secret (min 32 bytes) |
| `JWT_ISSUER` | — | Optional expected `iss` claim |
| `JWT_AUDIENCE` | — | Optional expected `aud` claim |
| `TLS_CERT_FILE` | — | Path to TLS certificate (enables TLS with KEY_FILE) |
| `TLS_KEY_FILE` | — | Path to TLS private key |
| `DB_QUERY_TIMEOUT` | 5s | Per-query context timeout |
| `DB_POOL_MAX_CONNS` | 10 | pgxpool max connections |
| `DB_POOL_MIN_CONNS` | 2 | pgxpool min idle connections |
| `DB_POOL_MAX_CONN_LIFETIME` | 3600s | Max connection reuse time |
| `DB_POOL_MAX_CONN_IDLE_TIME` | 1800s | Max connection idle time |
| `WHATSAPP_API_URL` | — | Base URL of the WhatsApp messaging API |
| `WHATSAPP_API_TOKEN` | — | Bearer token for WhatsApp API authentication |

## Database

Postgres 15. Migrations are plain SQL files in `internal/infrastructure/postgres/migrations/`, applied by a custom runner (`cmd/migrate/`) that tracks versions in a `schema_migrations` table. Each migration runs in a transaction.

Migration files follow the `{version}_{name}.up.sql` / `{version}_{name}.down.sql` naming convention. `make migrate` runs all pending `.up.sql` files in order; `make migrate-down` rolls back the last applied version using its `.down.sql` file.

## CI

`.github/workflows/ci.yml` runs on every push and pull request:

- **lint** job: `golangci-lint` with `.golangci.yml` config
- **test** job: `go test -race -coverprofile -covermode=atomic` on `internal/` packages (excludes `cmd/` composition roots), enforces ≥ 80% total coverage, writes per-function summary to the GitHub Actions step summary, uploads `coverage.out` as an artifact

To enforce CI before deploy: enable **branch protection rules** on `main` → *Require status checks to pass* → select `Lint` and `Test & Coverage`.
