# Changelog

All notable changes to this project will be documented in this file.

## [Unreleased]

### Added
- Go module initialization with core dependencies.
- Gin HTTP adapter with `POST /api/events`.
- Health check endpoint `GET /healthz`.
- Hexagonal structure wiring (`core`, `ports`, `adapters`, `app`).
- Postgres repository for events and SQL migration.
- HTTP handler tests for idempotency validation.
- JWT authentication middleware for `/api/v1/events`.
- Unit tests for domain validation and use case.
- Integration test for Postgres repository (tagged `integration`).
- Lint configuration via `golangci-lint` and Makefile targets.
- Security middlewares for request ID, JSON Content-Type, body size limit, and rate limiting.
- Configurable HTTP timeouts and request limits via environment variables.
- `.env.example` with recommended defaults.
- SQL safety helpers for dynamic identifiers.
- Additional event payload validations for PasswordResetRequested, OrderPaid, and OrderShipped.
- HTTP tests for Content-Type enforcement and rate limiting.
- `gosec` enabled via `golangci-lint`.
- Migration runner and `make migrate` target for local development.
- Error mapping for request timeouts and richer error logging context (event_type, idempotency_key).
- Added error codes for unauthenticated, permission_denied, and timeout with HTTP mapping.
- JWT middleware now uses the standard error codes; added errmap unit tests.

### Changed
- HTTP server implementation switched to Gin.
- Enforced UUID format validation for `Idempotency-Key`.
- Documented idempotency behavior and curl examples.
- Added `JWT_SECRET` config for API auth.
- `make test-integration` loads `.env` automatically.
- HTTP server timeouts are configurable.
