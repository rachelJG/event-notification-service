# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build & Development Commands

```bash
make test                # Run unit tests (no DB required)
make test-integration    # Run integration tests (needs Postgres, loads .env)
make lint                # Run golangci-lint
make migrate             # Apply DB migrations (loads .env)
make docs                # Render API Blueprint docs to HTML
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
internal/domain/ports/             → Output port interfaces (EventRepository)
internal/application/ports/        → Input port interfaces (SubmitEventUseCase)
internal/pkg/apperror/             → Typed error taxonomy (AppError with codes like invalid_argument, conflict, etc.)
internal/application/usecases/     → Business logic (SubmitEvent use case, delegates to entities.NewEvent)
internal/application/validation/   → Payload shape validation per event type (application concern, not domain)
internal/infrastructure/http/      → Gin router, handler, DTOs, RouterOptions, HealthChecker interface, middleware stack, error-to-HTTP mapping
internal/infrastructure/postgres/  → EventRepository implementation using pgxpool
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
| Application | `validation.ValidateEvent` | Payload shape: required fields per event type, format rules (e.g. email contains `@`), amount > 0 |

The domain does **not** validate payload content — that would require importing `encoding/json`, breaking the zero-dependency rule. Payload bytes are opaque to the domain.

The use case calls `ValidateEvent` **before** `NewEvent`: validate input first, construct the domain object only when input is known to be valid.

## Key Patterns

- **Idempotency**: Enforced at DB level via `UNIQUE(idempotency_key, type)` index. INSERT uses `ON CONFLICT DO UPDATE SET id = events.id RETURNING id` to return the original ID on duplicates.
- **Error mapping**: `internal/pkg/apperror` defines error codes. `internal/infrastructure/http/errmap` maps them to HTTP status codes. Handler never sets HTTP status directly from business logic.
- **Middleware stack** (in order): Zap logger+recovery → request ID → security headers → Prometheus metrics → CORS → body limit → content-type enforcement → rate limiter. JWT auth is applied only to the events route.
- **SQL safety**: `internal/infrastructure/postgres/sqlsafe.go` provides allowlist-based identifier sanitization for any dynamic SQL.

## Testing Approach

- Unit tests use fake/stub implementations (e.g., `fakeRepo` in handler and use case tests). No mocking library.
- HTTP handler tests use `httptest.NewRecorder` with a real Gin router.
- Integration tests are in `internal/tests/` with the `integration` build tag.

### Unit test coverage

| Package | File | What it covers |
|---|---|---|
| `domain/entities` | `event_test.go` | `NewEvent` invariants: empty type, unsupported type, empty idempotency key, all valid types accepted |
| `application/validation` | `validate_event_test.go` | All 4 event types (valid + invalid fields + invalid JSON + invalid email + amount rules), empty type, unsupported type |
| `application/usecases` | `submit_event_test.go` | Success, payload error, repo error, missing idempotency key, empty type, unsupported type |
| `infrastructure/http` | `handler_test.go` | Missing/invalid idempotency key, success, use case error, unauthorized, wrong content-type, rate limit |
| `infrastructure/http/errmap` | `errmap_test.go` | AppError code → HTTP status mapping |

### Integration test coverage (`internal/tests/`, requires Postgres)

| File | What it covers |
|---|---|
| `postgres_event_repository_integration_test.go` | Repository layer: create, idempotency at DB level |
| `submit_event_integration_test.go` | Use case + repository: all event types persisted, idempotency end-to-end, same key different type, invalid payload not persisted, unsupported type rejected, empty idempotency key rejected |

## Configuration

All config is via environment variables (see `.env.example`). `JWT_SECRET` is required in production (`APP_ENV=production`). Config validation in `config.Validate()` checks logical consistency (e.g., CORS_ALLOW_ALL disallowed in prod, HSTS requires non-prod or explicit opt-in).

## Database

Postgres 15. Migrations are plain SQL files in `internal/infrastructure/postgres/migrations/`, applied by a custom runner (`cmd/migrate/`) that tracks versions in a `schema_migrations` table. Each migration runs in a transaction.
