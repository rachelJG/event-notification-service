# AGENT.md — Resumable Execution Plan

This file is the **single source of truth** for the current state of the production plan.
Any AI agent (Claude, GPT, Gemini, Copilot, etc.) should read this file at the start
of each session to know what to do without needing prior context.

> **Core rule:** After completing each task, update the "Status" column in this file
> to `DONE` and add the date. This way the next agent (or yourself in another session)
> knows exactly where to pick up.

---

## 1. Project Context

**What it is:** An Event Notification Service in Go with hexagonal architecture.
It receives events via REST API, persists them in Postgres, and an async worker
processes them to send email notifications.

**Key files to orient yourself:**
- `CLAUDE.md` — Architecture, commands, patterns, codebase conventions
- `PRODUCTION_TASKS.md` — Detailed plan with specs, SQL, interfaces, and reference code
- This file (`AGENT.md`) — Current state and what to do next

**Useful commands:**
```bash
make build          # Compile API binary
make build-worker   # Compile worker binary
make test           # Run unit tests
make lint           # Run linter
make test-integration  # Run integration tests (requires Postgres)
```

---

## 2. Task Status

Legend: `DONE` = completed, `TODO` = pending, `IN-PROGRESS` = someone started it

### Phase 1 — Data Model (migrations)

| ID       | Task                                  | Status | Date       |
|----------|---------------------------------------|--------|------------|
| TASK-1.1 | Migration: `notifications` table      | DONE   | 2026-02-23 |
| TASK-1.2 | Migration: `status` column on `events`| DONE   | 2026-02-23 |

### Phase 2 — Domain and Ports

| ID       | Task                                            | Status | Date       |
|----------|-------------------------------------------------|--------|------------|
| TASK-2.1 | `Notification` entity                           | DONE   | 2026-02-23 |
| TASK-2.2 | `NotificationRepository` port                   | DONE   | 2026-02-23 |
| TASK-2.3 | `EmailSender` port                              | DONE   | 2026-02-23 |
| TASK-2.4 | `ClaimPending`/`SetStatus` on `EventRepository` port | DONE | 2026-02-23 |

### Phase 3 — Use Cases

| ID       | Task                                  | Status | Date       |
|----------|---------------------------------------|--------|------------|
| TASK-3.1 | `ProcessEvents` use case              | DONE   | 2026-02-23 |
| TASK-3.2 | `DeliverNotifications` use case       | DONE   | 2026-02-23 |
| TASK-3.3 | Input ports for both use cases        | DONE   | 2026-02-23 |

### Phase 4 — Email Infrastructure

| ID       | Task                                  | Status | Date       |
|----------|---------------------------------------|--------|------------|
| TASK-4.1 | SMTP adapter                          | DONE   | 2026-02-23 |
| TASK-4.2 | Email templates per event type        | TODO   |            |
| TASK-4.3 | SMTP configuration in `config.go`     | DONE   | 2026-02-23 |

> **Note on TASK-4.2:** Email rendering is currently inline in the `ProcessEvents`
> use case (`renderEmail` function). The task requires extracting it to a dedicated
> `internal/infrastructure/email/templates.go` file.

### Phase 5 — Async Worker

| ID       | Task                                         | Status | Date       |
|----------|----------------------------------------------|--------|------------|
| TASK-5.1 | `cmd/worker/main.go`                         | DONE   | 2026-02-23 |
| TASK-5.2 | `NotificationRepository` Postgres impl       | DONE   | 2026-02-23 |
| TASK-5.3 | `ClaimPending`/`SetStatus` implementation    | DONE   | 2026-02-23 |
| TASK-5.4 | Worker in Docker/docker-compose              | DONE   | 2026-02-23 |
| TASK-5.5 | Makefile targets for worker                  | DONE   | 2026-02-23 |

### Phase 6 — Production Hardening

| ID       | Task                                         | Status | Date       |
|----------|----------------------------------------------|--------|------------|
| TASK-6.1 | Rate limiter with `X-Forwarded-For` support  | DONE   | 2026-02-23 |
| TASK-6.2 | `Retry-After` header on 429 responses        | DONE   | 2026-02-23 |
| TASK-6.3 | Return original ID in 409 Conflict response  | DONE   | 2026-02-23 |
| TASK-6.4 | `Cache-Control: no-store` header on API routes | DONE | 2026-02-23 |
| TASK-6.5 | Validate positive values in config           | DONE   | 2026-02-23 |
| TASK-6.6 | Configurable shutdown timeout                | DONE   | 2026-02-23 |

### Phase 7 — Observability

| ID       | Task                                         | Status | Date       |
|----------|----------------------------------------------|--------|------------|
| TASK-7.1 | pgxpool metrics as Prometheus collector       | TODO   |            |
| TASK-7.2 | HTTP error counter by code                   | TODO   |            |
| TASK-7.3 | Worker metrics                               | TODO   |            |
| TASK-7.4 | Audit logging in JWT middleware              | TODO   |            |

### Phase 8 — Testing

| ID       | Task                                         | Status | Date       |
|----------|----------------------------------------------|--------|------------|
| TASK-8.1 | Tests for `ProcessEvents`/`DeliverNotifications` | DONE | 2026-02-23 |
| TASK-8.2 | SMTP adapter unit tests                      | DONE   | 2026-02-23 |
| TASK-8.3 | Health endpoint tests                        | TODO   |            |
| TASK-8.4 | Worker integration test                      | TODO   |            |
| TASK-8.5 | Raise coverage threshold to 60%              | TODO   |            |

### Phase 9 — Deployment

| ID       | Task                                         | Status | Date       |
|----------|----------------------------------------------|--------|------------|
| TASK-9.1 | Kubernetes manifests                         | TODO   |            |
| TASK-9.2 | Complete deploy workflow in CI               | TODO   |            |
| TASK-9.3 | Worker build step in CI                      | TODO   |            |

---

## 3. What To Do Next

**Next task:** The first `TODO` task in phase order.
Currently: **TASK-4.2** (email templates).

**Recommended order:**
1. TASK-4.2 — Only remaining task from the core phases (1-5)
2. Phase 7 — Observability (TASK-7.3 depends on the existing worker)
4. Phase 8 — Testing for new features (TASK-8.3, 8.4 after functionality is in place, 8.5 last)
5. Phase 9 — Deployment (last, when everything else is ready)

**Cross-phase dependencies:**
- Phases 1-6: Completed (core pipeline + production hardening)
- Phase 7.3: Depends on Phase 5 (DONE)
- Phase 8.4: Depends on Phases 3+5 (DONE)
- Phase 8.5: Do last, after all tests are added
- Phase 9: Do last

---

## 4. Agent Instructions

### Starting a session
1. Read `AGENT.md` (this file) to understand current state
2. Read `CLAUDE.md` to understand the architecture and conventions
3. Find the first `TODO` task in Section 2
4. Read the detailed spec for that task in `PRODUCTION_TASKS.md`
5. Implement the task following codebase conventions
6. Run `make test` and `make lint` to validate
7. Update this file: change `TODO` to `DONE` and add the date

### Code conventions (quick reference)
- **Hexagonal architecture:** domain never imports infrastructure
- **Errors:** use `apperror.AppError` with defined codes
- **Tests:** fakes/stubs, no mocking libraries
- **SQL:** use `sqlsafe` for dynamic identifiers
- **Config:** everything via env vars, validate in `config.Validate()`

### After completing a task
- Mark the task as `DONE` in the table above
- If the task generated additional work, add a note below the relevant table
- If a task is partially done, mark it as `IN-PROGRESS` and add a note
  explaining what remains

### If something goes wrong
- Do NOT delete or modify `PRODUCTION_TASKS.md` (it is the immutable reference)
- Add notes in this file below the affected table
- If you discover a problem with the spec, document it here for the next session
