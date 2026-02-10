# Changelog

All notable changes to this project will be documented in this file.

## [Unreleased]

### Added
- Go module initialization with core dependencies.
- Gin HTTP adapter with `POST /api/events`.
- Hexagonal structure wiring (`core`, `ports`, `adapters`, `app`).
- Postgres repository for events and SQL migration.
- Unit tests for domain validation and use case.
- Integration test for Postgres repository (tagged `integration`).
- Lint configuration via `golangci-lint` and Makefile targets.

### Changed
- HTTP server implementation switched to Gin.
