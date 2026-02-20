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
go test ./internal/domain/entities/ -run TestValidateEvent
```

Integration tests use the `//go:build integration` tag and require a live Postgres instance (`PG_DSN` env var).

## Architecture

Hexagonal Architecture (Domain / Application / Infrastructure). The domain layer has zero external dependencies.

```
cmd/api/main.go                    → Entry point, composition root, signal handling, graceful shutdown
internal/config/                   → Env-var-based configuration (shared, layer-independent)
internal/domain/entities/          → Event entity, payload types, validation (no imports from infrastructure)
internal/domain/ports/             → Port interfaces (EventRepository, SubmitEventUseCase)
internal/domain/errors/            → Typed error taxonomy (AppError with codes like invalid_argument, conflict, etc.)
internal/application/usecases/     → Business logic (SubmitEvent use case)
internal/infrastructure/http/      → Gin router, handler, DTOs, middleware stack, error-to-HTTP mapping
internal/infrastructure/postgres/  → EventRepository implementation using pgxpool
internal/infrastructure/logger/    → Zap logger factory
```

**Dependency flow:** `cmd/api (composition root) → infrastructure → ports ← usecases ← entities`. Infrastructure depends on ports; domain and application never import infrastructure.

## Key Patterns

- **Idempotency**: Enforced at DB level via `UNIQUE(idempotency_key, type)` index. INSERT uses `ON CONFLICT DO UPDATE SET id = events.id RETURNING id` to return the original ID on duplicates.
- **Error mapping**: `internal/domain/errors` defines domain error codes. `internal/infrastructure/http/errmap` maps them to HTTP status codes. Handler never sets HTTP status directly from business logic.
- **Middleware stack** (in order): Zap logger+recovery → request ID → security headers → Prometheus metrics → CORS → body limit → content-type enforcement → rate limiter. JWT auth is applied only to the events route.
- **SQL safety**: `internal/infrastructure/postgres/sqlsafe.go` provides allowlist-based identifier sanitization for any dynamic SQL.

## Testing Approach

- Unit tests use fake/stub implementations (e.g., `fakeRepo` in handler and use case tests). No mocking library.
- HTTP handler tests use `httptest.NewRecorder` with a real Gin router.
- Integration tests are in `internal/tests/` with the `integration` build tag.

## Configuration

All config is via environment variables (see `.env.example`). `JWT_SECRET` is required in production (`APP_ENV=production`). Config validation in `config.Validate()` checks logical consistency (e.g., CORS_ALLOW_ALL disallowed in prod, HSTS requires non-prod or explicit opt-in).

## Database

Postgres 15. Migrations are plain SQL files in `internal/infrastructure/postgres/migrations/`, applied by a custom runner (`cmd/migrate/`) that tracks versions in a `schema_migrations` table. Each migration runs in a transaction.
