# Backend Development Best Practices

> **Last updated:** 2026-03-11
>
> This guide establishes quality, maintainability, and scalability standards for backend projects. A separate document exists for frontend.

---

## 1. Version Control (Commits and Branches)

*   **Conventional Commits**: Mandatory for readable history and automation (CHANGELOG and SemVer).
    *   `feat:`: New feature. E.g.: `feat: add billing endpoint`.
    *   `fix:`: Bug fix. E.g.: `fix: resolve timeout in payment query`.
    *   `docs:`: Documentation update. E.g.: `docs: update readme`.
    *   `chore:`, `refactor:`, `test:`, `ci:`, `perf:` for other tasks.
    *   `BREAKING CHANGE:`: Change that breaks backward compatibility (bumps major version).
*   **Semantic Versioning (SemVer)**: Git version must follow `MAJOR.MINOR.PATCH`. Tools like `semantic-release` or `release-please` automate this from commit messages.
*   **Atomic Commits**: One commit = one logical change. Code must compile and tests must pass at every commit in the history.
*   **Branching Strategy (GitHub Flow / Trunk-based)**:
    *   The `main` branch is always protected and deployable.
    *   No one pushes directly to `main`. Everything enters through Pull Requests (PR) from short-lived branches.
*   **Merge Strategy**: Choose per team and maintain consistency:
    *   **Squash and Merge**: Condenses all PR commits into one. Ideal when intermediate commits are noisy (WIP, fixups). Clean history on `main`.
    *   **Rebase and Merge**: Preserves each individual commit. Preferable when commits are already atomic and descriptive. Maintains a linear and detailed history.
    *   What matters is that the team picks a strategy and applies it consistently.

---

## 2. Code Review

Code review is a mandatory practice. Every change goes through review before reaching `main`.

*   **Minimum 1 approval** before merging. For critical systems, consider 2.
*   **CODEOWNERS**: Configure `.github/CODEOWNERS` to auto-assign reviewers by code area (e.g.: `internal/infrastructure/postgres/ @data-team`).
*   **What to review** (in priority order):
    1.  **Correctness**: Does it do what it claims? Does it handle errors properly?
    2.  **Security**: Does it introduce vulnerabilities? Does it expose sensitive data?
    3.  **Design**: Does it respect the architecture? Are abstractions appropriate?
    4.  **Readability**: Can it be understood without verbal explanation?
    5.  **Performance**: Are there N+1 queries, unnecessary allocations, or blocking calls?
*   **Reviewer best practices**:
    *   Constructive comments: explain the "why", not just the "what".
    *   Distinguish between blockers (must fix) and suggestions (nice to have).
    *   Don't block PRs for style issues if a linter handles them.
*   **Author best practices**:
    *   Small, focused PRs (< 400 lines ideally).
    *   Clear PR description: what changes, why, how to test it.
    *   Self-review the diff before requesting review.

---

## 3. README Structure

The `README.md` is the project's front page. Recommended structure:

1.  **Title and Badges**: CI status, coverage, version.
2.  **Description**: What the project does and why it exists.
3.  **Overview**: Key features, tech stack, and architecture diagram.
4.  **Prerequisites**: Required dependencies (e.g.: Go 1.22, Docker, PostgreSQL 15).
5.  **Installation and Setup**: Concrete steps (e.g.: copy `.env.example` to `.env`).
6.  **Running Locally**: Commands to start the app (e.g.: `make run`, `docker-compose up`).
7.  **Project Structure**: Brief explanation of key directories.
8.  **Running Tests**: Commands for running tests and linters.
9.  **Usage / Endpoints**: `curl` examples or link to OpenAPI/Swagger.

---

## 4. Configuration Management

Application configuration is everything that varies between environments (development, staging, production): DB credentials, ports, external service URLs, feature flags, etc. Follow the [Twelve-Factor App: Config](https://12factor.net/config) principle.

### 4.1 Golden Rule: Configuration Outside the Code

*   Configuration is injected via **environment variables**, never hardcoded in source code.
*   If a value changes between environments, it's configuration. If it's the same across all environments, it's a code constant.

### 4.2 `.env` Files — Local Development Only

*   The `.env` file is a convenience for local development. It **must never** exist on production servers.
*   **`.env`** → Contains actual values for your local machine. **Never commit it** (must be in `.gitignore`).
*   **`.env.example`** → Committed to the repository. Serves as living documentation of all variables the application needs. Contains example or empty values, never real secrets.
    ```env
    # .env.example
    API_ADDR=:8080
    PG_DSN=postgres://postgres:postgres@localhost:5432/mydb?sslmode=disable
    LOG_LEVEL=info
    APP_ENV=development

    # Secrets (leave empty, each dev fills in their own)
    SMTP_PASSWORD=
    API_KEY_SECRET=
    ```
*   When cloning the project, the first step is: `cp .env.example .env` and fill in the missing values.

### 4.3 Configuration in Production

In production, environment variables are injected from the orchestrator or platform, **not from `.env` files**:

| Platform | How variables are configured |
|----------|------------------------------|
| **Kubernetes** | `ConfigMap` (non-sensitive) + `Secret` (sensitive), mounted as env vars in the pod |
| **Docker Compose** | `environment:` section in `docker-compose.yml` or `env_file:` directive |
| **AWS ECS/Fargate** | Task Definition → environment + Secrets Manager references |
| **Railway / Render / Fly.io** | Platform dashboard → Environment variables |
| **Systemd** | `Environment=` or `EnvironmentFile=` in the unit file |

### 4.4 Separate Configuration from Secrets

Not everything is equally sensitive. Distinguish between:

*   **Non-sensitive configuration**: Ports, log levels, timeouts, pool sizes, feature flags. Can live in `ConfigMap`, configuration files, or environment variables without additional protection.
*   **Secrets**: DB passwords, API keys, tokens, TLS certificates. Must be stored in dedicated systems:
    *   Kubernetes Secrets (encrypted at rest with KMS)
    *   HashiCorp Vault
    *   AWS Secrets Manager / GCP Secret Manager / Azure Key Vault
*   **Never** log secrets. Sanitize sensitive variables before printing them in startup logs.

### 4.5 Validation at Startup

*   Validate **all** configuration when the application starts. If a required variable is missing or has an invalid value, the app must **fail immediately** with a clear message.
    ```
    invalid config: PG_DSN is required
    invalid config: RATE_LIMIT_RPS must be > 0
    ```
*   Don't wait for the first request to fail to discover missing configuration.
*   Validate coherence: if `HSTS_ENABLED=true`, then `HSTS_MAX_AGE_SECONDS` must be > 0. If `TLS_CERT_FILE` is set, `TLS_KEY_FILE` must be set too.
*   Provide **sensible defaults** for development: the app should start with `cp .env.example .env` without modifying anything for the most basic case.

### 4.6 Recommended Code Structure

```
// Centralize in a single package (e.g.: internal/config/)
config.Load()      → Reads all environment variables with defaults
config.Validate()  → Validates coherence and required fields
```

*   A single place where configuration is read, not `os.Getenv()` scattered throughout the code.
*   Internal layers receive only the values they need (specific structs or individual fields), not the entire `Config` object. This respects dependency inversion.

---

## 5. Security

Security is not optional. It must be present from the design phase, not bolted on at the end.

### 5.1 Secret Management
*   **Never** hardcode credentials, tokens, or keys in source code (see [Section 4.4](#44-separate-configuration-from-secrets)).
*   For environments with multiple secrets, use a dedicated manager (HashiCorp Vault, AWS Secrets Manager, GCP Secret Manager).
*   Add `.env` to `.gitignore` (see [Section 4.2](#42-env-files--local-development-only)).

### 5.2 Dependencies
*   Maintain a **lock file** (`go.sum`, `package-lock.json`, `poetry.lock`) and commit it to the repository.
*   Run vulnerability scans periodically and manually: `govulncheck ./...` (Go), `npm audit` (Node), `pip-audit` (Python).
*   Before updating dependencies, check the changelog and run the full test suite. Updates must be deliberate, not automatic.
*   Before adding a new dependency, evaluate: Is it actively maintained? What's its license? How many transitive dependencies does it bring?

### 5.3 OWASP Top 10 (Most Relevant for Backend)
*   **Injection (SQL, NoSQL, OS)**: Always use parameterized queries or prepared statements. Never interpolate user input into queries.
*   **Broken Authentication**: Hash passwords with bcrypt/argon2 (never MD5/SHA1 for passwords). Implement rate limiting on login endpoints.
*   **Sensitive Data Exposure**: Don't return internal data in error responses (stack traces, SQL queries, filesystem paths).
*   **Broken Access Control**: Validate permissions on the backend, never trust client-side validations. Apply the principle of least privilege.
*   **Security Misconfiguration**: Disable debug endpoints in production. Configure security headers (HSTS, Content-Security-Policy, X-Content-Type-Options).

### 5.4 Authentication and Authorization

There are two main mechanisms for protecting APIs. They are not interchangeable — each solves a different problem.

#### JWT (JSON Web Token) — User Identity

*   **When to use it**: When you need to know **who** the user is (login, sessions, per-user permissions).
*   **How it works**: The client authenticates (login), receives a signed token containing claims (`sub`, `exp`, `roles`). The backend validates the signature without querying the DB on each request.
*   **Standard header**: `Authorization: Bearer <token>` (RFC 6750, OAuth 2.0 standard).
    ```
    Authorization: Bearer eyJhbGciOiJIUzI1NiIs...
    ```
*   **Best practices**:
    *   Secure algorithm: HS256 (shared secret) or RS256/ES256 (asymmetric key).
    *   Always require `exp` (expiration). Short-lived tokens (15-60 min) + refresh token.
    *   Validate `iss` (issuer) and `aud` (audience) if the system has multiple services.
    *   Minimum 32-byte secret for HS256.
    *   **Never** store sensitive data in the payload — JWT is signed, not encrypted (anyone can read the contents with base64).
*   **Typical use cases**: Apps with users (SaaS, e-commerce, social networks), SPAs, mobile apps.

#### API Key — System/Service Identity

*   **When to use it**: **Service-to-service** communication, third-party integrations, public APIs with usage plans, webhooks.
*   **How it works**: A unique key is generated per client/service. The backend looks up the key's hash in the DB to validate it and determine its permissions (scopes).
*   **Standard header**: `X-API-Key: <key>` (industry convention: AWS, Stripe, Google Cloud, Twilio).
    ```
    X-API-Key: ens_a1b2c3d4e5f6...
    ```
*   **Do NOT use** `Authorization: Bearer` for API keys — that scheme is for OAuth 2.0 tokens (RFC 6750). Mixing them causes confusion and breaks the standard's semantics.
*   **Best practices**:
    *   Store only the **SHA-256 hash** of the key, never the plaintext value.
    *   Show the full key **only once** at creation time.
    *   Implement **scopes** (granular permissions): `events:read`, `events:write`, `admin`.
    *   Allow **revoking** keys individually without affecting other clients.
    *   Use an identifiable prefix (e.g.: `ens_`, `sk_`, `pk_`) to facilitate leak detection in logs and scans.
    *   Track `last_used_at` to identify inactive keys.
*   **Typical use cases**: Public APIs (Stripe, Twilio), B2B integrations, CI/CD pipelines, internal microservices.

#### When to Use Which?

| Scenario | Mechanism | Reason |
|----------|-----------|--------|
| Web/mobile app with user login | JWT | You need user identity and sessions |
| Public API for third parties | API Key | The consumer is a system, not a human |
| Microservice A calls microservice B | API Key or mTLS | Service identity, not user identity |
| Webhook receiving external events | API Key + HMAC signature | Validate origin and payload authenticity |
| Internal API + users with roles | JWT | Granular per-user permissions |
| Admin dashboard accessed by people | JWT | User session with roles |

In some systems both are combined: **API Key** to identify the client/tenant + **JWT** to identify the user within that tenant.

### 5.5 Principle of Least Privilege
*   Database users with only the necessary permissions (don't use the superuser for the application).
*   CI/CD tokens scoped to the specific repository or action.
*   Service accounts with granular permissions, not admin roles.

---

## 6. Testing (Testing Pyramid)

Percentages are guidelines and should be adjusted based on the project type. A service with little pure logic may benefit more from integration tests than unit tests.

*   **Unit Tests (~70%)**:
    *   Test isolated business logic (pure functions, validations, transformations).
    *   Use Mocks/Stubs to isolate external dependencies (DB, APIs, filesystem).
    *   Must be fast (< 1s per test) and deterministic (no external state dependency).
*   **Integration Tests (~25%)**:
    *   Validate real interaction with the database, caches, message queues.
    *   **Best practice**: Spin up real dependencies in ephemeral Docker containers (Testcontainers or docker-compose in CI).
    *   Verify that SQL queries, migrations, and mappings work against the real engine.
*   **End-to-End Tests (~5%)**:
    *   Test the complete flow: HTTP request → logic → database → HTTP response.
    *   Reserve for critical business flows (e.g.: create order, process payment).
    *   They are the slowest and most brittle; minimize their count.
*   **Minimum coverage**: Set a threshold (e.g.: 80%) in CI and enforce it. Coverage doesn't guarantee quality, but its absence guarantees blind spots.
*   **Race detector**: In languages that support it (Go: `-race`), always run it in CI to detect race conditions.

---

## 7. Error Handling

Errors are first-class citizens. Poor error handling is the most common cause of hard-to-diagnose bugs in production.

*   **Never silence errors**: Don't use `_ = doSomething()` without documented justification.
*   **Typed errors**: Define an application error taxonomy (e.g.: `NotFound`, `Conflict`, `InvalidArgument`, `Internal`). This allows each layer to react appropriately.
*   **Propagate with context**: Wrap errors with additional information when propagating. In Go: `fmt.Errorf("create user: %w", err)`. In other languages, use exceptions with descriptive messages.
*   **Separate business errors from infrastructure errors**: "User not found" is not the same as "the database is not responding". The former is expected; the latter is a failure.
*   **Centralized HTTP mapping**: The HTTP layer translates business errors to HTTP codes in a single place. Handlers must never choose the status code directly from business logic.
*   **Retries with external services**: Use exponential backoff with jitter. Define a maximum number of retries. Implement circuit breaker if the service fails persistently.

---

## 8. Code Documentation

*   **Self-documenting code**: Clear names for variables, functions, and methods. The code says the "what".
*   **"Why" comments**: Explain unusual decisions, workarounds, or business constraints. Don't repeat what the code already says.
*   **Architecture Decision Records (ADR)**: Document important technical decisions in `docs/adr/` (e.g.: `001-use-of-kafka.md`, `002-migration-to-api-keys.md`). Format: context, decision, consequences.
*   **API contracts**: Use **OpenAPI/Swagger** as mandatory for documenting endpoints. It's the contract between the backend and its consumers. Keep it in sync with the code.
*   **Document exported functions and types**: In Go, every exported symbol must have a comment starting with the symbol's name. In other languages, follow the ecosystem convention (JSDoc, docstrings, etc.).

---

## 9. Observability

A system that cannot be observed cannot be operated. Observability is built on three pillars:

### 9.1 Logs
*   **Structured JSON** format to `stdout`/`stderr` (never write directly to files).
*   Always include: `timestamp`, `level`, `message`, `request_id`/`trace_id`.
*   Levels with purpose: `DEBUG` for development, `INFO` for normal flow, `WARN` for recoverable situations, `ERROR` for failures requiring attention.
*   Don't log sensitive data (tokens, passwords, personal data).

### 9.2 Metrics
*   Expose metrics in **Prometheus** format (`/metrics`).
*   RED metrics for every service: **Rate** (requests/s), **Errors** (error rate), **Duration** (latency p50, p95, p99).
*   Business metrics: events processed, payments completed, etc. Separate from infrastructure metrics.
*   Resource metrics: DB connection pool, active goroutines, memory usage.

### 9.3 Distributed Tracing
*   Implement **OpenTelemetry** as the standard for tracing.
*   Propagate `trace_id` across all inter-service calls.
*   Instrument: HTTP handlers, DB queries, external service calls.
*   Tracing allows visualizing the complete flow of a request across multiple services.

### 9.4 Alerting
*   Alert on **symptoms** (high latency, elevated error rate), not causes (CPU at 80%).
*   Define **SLIs** (Service Level Indicators) and **SLOs** (Service Level Objectives) for critical services.
*   Every alert must have an associated runbook: what it means, how to diagnose, how to remediate.
*   Avoid alert fatigue: if an alert is consistently ignored, adjust or remove it.

---

## 10. REST API Standards

### 10.1 Design
*   **Versioning**: URL prefix (`/api/v1/`, `/api/v2/`). Never break an existing contract without a new version.
*   **Resources as nouns**: `/api/v1/users`, not `/api/v1/getUsers`.
*   **Correct HTTP verbs**: `GET` (read), `POST` (create), `PUT` (replace), `PATCH` (partial update), `DELETE` (delete).
*   **Consistent status codes**:
    *   `200` OK, `201` Created, `202` Accepted (async operations)
    *   `400` Bad Request, `401` Unauthorized, `403` Forbidden, `404` Not Found, `409` Conflict, `429` Too Many Requests
    *   `500` Internal Server Error, `503` Service Unavailable, `504` Gateway Timeout

### 10.2 Structured Error Responses
*   Consistent format across all endpoints:
    ```json
    {
      "error": "human-readable description for the developer",
      "code": "INVALID_ARGUMENT",
      "request_id": "abc-123"
    }
    ```
*   Never expose internal details (stack traces, table names, SQL queries) in error responses.

### 10.3 Pagination, Filtering, and Sorting
*   **Pagination**: Use cursor-based (`?cursor=abc&limit=20`) for large datasets, or offset-based (`?page=1&per_page=20`) for small and predictable datasets.
*   **Filtering**: Descriptive query params (`?status=active&type=payment`).
*   **Sorting**: `?sort=created_at&order=desc`.
*   Document allowed values and defaults.

### 10.4 Idempotency
*   `POST` operations that create resources should support an `Idempotency-Key` header to prevent duplicates on client retries.
*   `PUT` and `DELETE` are idempotent by definition.

---

## 11. Architecture and Structure

*   **Clean Architecture**: For enterprise projects, use an architecture that strictly separates business logic from infrastructure (Hexagonal, Onion, Clean Architecture). The key principle is **dependency inversion**: the domain does not depend on frameworks, databases, or HTTP transport.
*   **Layered folder structure**:
    ```
    cmd/            → Entry points (main.go)
    internal/
      domain/       → Entities, value objects, output port interfaces
      application/  → Use cases, validation, input port interfaces
      infrastructure/ → Implementations: HTTP, DB, messaging, external services
      pkg/          → Shared utilities within the project
    ```
*   **Don't expose internals**: Use `internal/` to prevent other modules from importing packages they shouldn't.
*   **Dependency injection at the composition root**: All object creation and wiring happens in `cmd/*/main.go`. Internal layers receive their dependencies as interfaces.

---

## 12. Continuous Integration (CI)

*   **Mandatory on every Pull Request**:
    *   Linter (`golangci-lint`, `ruff`, `eslint`, etc.)
    *   Unit tests with race detector where applicable
    *   Minimum coverage verification
    *   Dependency vulnerability scanning
*   **Block failing PRs**: Configure branch protection on `main` to require all checks to pass.
*   **Reproducible builds**: Pin tool versions in CI (Go, linters, etc.). Use lock files.
*   **Artifacts**: Store coverage reports and binaries as pipeline artifacts.

---

## 13. Continuous Deployment (CD)

*   **Deployment strategies**:
    *   **Rolling update**: Replaces instances gradually. Simple and suitable for most services.
    *   **Blue-Green**: Two identical environments; you switch traffic from one to the other. Instant rollback.
    *   **Canary**: Sends a small percentage of traffic to the new version. Allows detecting issues before a full rollout.
*   **Health checks**: Every service must expose endpoints that the orchestrator uses to determine the instance's state. In **Kubernetes**, there are 3 probes that must be implemented:

    | Probe | Suggested Endpoint | Question It Answers | If It Fails... |
    |-------|-------------------|---------------------|-----------------|
    | **Liveness** | `GET /healthz` | Is the process alive and not hung? | K8s **restarts** the pod |
    | **Readiness** | `GET /readyz` | Can it receive traffic? (DB connected, migrations applied) | K8s **stops sending** requests (removes it from the Service, but doesn't restart it) |
    | **Startup** | `GET /healthz` | Has it finished starting up? | K8s waits; if it doesn't pass within the configured time, restarts the pod |

    *   **Liveness** (`/healthz`): Must be lightweight and fast. Only verifies the process responds. **Do not** check external dependencies here — if the DB goes down, you don't want K8s to restart all pods in a cascade.
    *   **Readiness** (`/readyz`): Here you do check dependencies: DB connection, caches, critical services. If the DB is not responding, the pod is marked as "not ready" and stops receiving traffic, but stays alive waiting for the dependency to recover.
    *   **Startup**: Useful for services with slow startup (warming caches, loading models). Uses the same endpoint as liveness but with a more generous timeout (`failureThreshold * periodSeconds`).
    *   Health endpoints **do not require authentication**.
    *   Example Kubernetes deployment configuration:
        ```yaml
        livenessProbe:
          httpGet:
            path: /healthz
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 10
          failureThreshold: 3
        readinessProbe:
          httpGet:
            path: /readyz
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 5
          failureThreshold: 3
        startupProbe:
          httpGet:
            path: /healthz
            port: 8080
          failureThreshold: 30
          periodSeconds: 2
        ```
*   **Rollback**: It must be possible to revert to the previous version in under 5 minutes. If it's not, the deployment process needs improvement.
*   **Database migrations**: Must be forward-compatible. Deploy the new schema first, then the code that uses it. Never make destructive changes (DROP COLUMN) in the same deploy that changes the code.
*   **Feature Flags**: Use feature flags to decouple deploy from release. Allows enabling/disabling features without re-deploy and instant rollback of specific features.

---

## 14. Dependency Management

*   **Lock files are mandatory**: Always commit `go.sum`, `package-lock.json`, `poetry.lock`, etc. Guarantees reproducible builds.
*   **Deliberate updates**: Dependencies are updated manually and consciously, not automatically. Recommended process:
    1.  Review the changelog of the new version
    2.  Check for breaking changes
    3.  Update in a dedicated branch
    4.  Run the full test suite
    5.  Review and merge like any other PR
*   **Evaluate before adding**: Before adding a new dependency, ask:
    *   Is it actively maintained? (last commit, open issues)
    *   What's its license? (MIT, Apache 2.0 are safe; GPL can be problematic)
    *   How many transitive dependencies does it bring?
    *   Can it be solved with custom code in under 50 lines?
*   **Vulnerability scanning**: Periodically run `govulncheck ./...` (Go), `npm audit` (Node) or the ecosystem equivalent. Integrate into CI as a non-blocking check initially, blocking when the team is ready.

---

## 15. Database

*   **Versioned migrations**: Use numbered SQL files (`001_create_users.up.sql` / `001_create_users.down.sql`). Each migration runs within a transaction.
*   **Always have a down migration**: Every `up` migration must have a corresponding `down` to allow reverting.
*   **Never modify already-applied migrations**: If you need to change something, create a new migration.
*   **Connection pooling**: Configure the connection pool with explicit limits (`max_conns`, `min_conns`, `max_conn_lifetime`, `max_conn_idle_time`). Monitor pool usage.
*   **Per-query timeouts**: Use `context.WithTimeout` on every query. A query without a timeout is a cascading failure risk.
*   **Indexes**: Create indexes for frequently executed queries. Use `EXPLAIN ANALYZE` to validate they are being used correctly.
*   **Sensitive data**: Hash what you don't need to read (passwords, API keys). Encrypt what you need to read but not in plaintext (personal data, depending on compliance).

---

## 16. Production-Ready API — Middlewares and Validation

An API that works in development is not ready for production. This section details the **minimum middlewares** every backend API must have before going to production, the **types of input validation**, and other aspects that are often overlooked.

### 16.1 Middleware Stack

Middlewares execute in order for each request. **Order matters** — for example, the request ID must be assigned before the logger so it appears in all logs.

Below are all common middlewares, indicating which are **mandatory** for production and which are **recommended** depending on the use case.

#### 1. Recovery / Panic Handler — MANDATORY

*   **What it does**: Catches unhandled panics inside a handler and returns a `500 Internal Server Error` instead of crashing the entire process.
*   **Why it's needed**: Without this middleware, a panic in a handler (nil pointer, index out of range) kills the entire process and affects all users, not just that request.
*   **Example**: A handler accesses `user.Address.City` but `Address` is nil → panic → without recovery, the server dies.
*   **Note**: Having recovery doesn't mean panics are acceptable. Every caught panic must be logged as `ERROR` with a stack trace for investigation.

#### 2. Request ID — MANDATORY

*   **What it does**: Assigns a unique identifier (UUID v4 or similar) to each incoming request. Propagates it in the context and includes it in the response header (`X-Request-ID`).
*   **Why it's needed**: Without a request ID, correlating logs between services or with the client is impossible. When a user reports an error, the request ID is what allows finding the complete trace.
*   **Behavior**: If the client already sends an `X-Request-ID` (e.g., a gateway or load balancer), reuse it instead of generating a new one. This maintains end-to-end traceability.

#### 3. Logger — MANDATORY

*   **What it does**: Logs each incoming request and its response: method, path, status code, latency, client IP, request ID.
*   **Why it's needed**: It's the primary source of information for debugging, auditing, and traffic analysis. Without per-request logs, diagnosing production issues is guesswork.
*   **Must execute after**: Recovery (to log panics) and Request ID (to include it in every log).
*   **What NOT to log**: Full request/response bodies (volume and privacy), authentication headers, personal data.

#### 4. Security Headers — MANDATORY

*   **What it does**: Adds HTTP headers that protect against common browser attacks.
*   **Why it's needed**: Even if the backend doesn't serve HTML directly, these protections prevent responses from being misinterpreted by the browser.
*   **Minimum headers**:

    | Header | Value | Protects Against |
    |--------|-------|------------------|
    | `X-Content-Type-Options` | `nosniff` | Browser guessing MIME type and executing malicious content |
    | `X-Frame-Options` | `DENY` or `SAMEORIGIN` | Clickjacking (embedding your site in an iframe) |
    | `X-XSS-Protection` | `0` | Disables the browser's XSS filter (can cause more issues than it solves in modern browsers) |
    | `Strict-Transport-Security` (HSTS) | `max-age=31536000; includeSubDomains` | Man-in-the-middle by forcing HTTPS. **Only enable if the service already uses TLS** |
    | `Content-Security-Policy` | `default-src 'none'` | Script injection. For pure APIs that only return JSON, this value is sufficient |
    | `Referrer-Policy` | `no-referrer` | URL leakage in the Referer header |

*   **Is HSTS mandatory?**: Only if your service is already behind TLS (directly or through a load balancer). If you serve plain HTTP in development, HSTS must not be active.
*   **CSP on an API?**: Yes, with `default-src 'none'`. If someone accesses an error endpoint from a browser, CSP prevents content injection.

#### 5. Metrics (Prometheus) — MANDATORY

*   **What it does**: Automatically measures each request: request counter, latency histogram, broken down by method, path, and status code.
*   **Why it's needed**: Without metrics there are no dashboards, no alerts, no SLOs. It's the foundation of all quantitative observability.
*   **Must be a middleware**: This way it covers all endpoints without having to instrument each handler manually.
*   **Watch out for cardinality**: Don't use paths with dynamic IDs as labels (`/users/12345`). Normalize to `/users/:id` or the metrics will explode in cardinality.

#### 6. CORS (Cross-Origin Resource Sharing) — CONDITIONAL

*   **What it does**: Controls which origins (domains) can make requests to the API from a browser.
*   **When it's needed**: When the API is consumed by a frontend on a different domain (e.g., `app.example.com` consumes `api.example.com`).
*   **When it's NOT needed**: APIs consumed exclusively by backends (service-to-service), native mobile apps, or CLIs. The browser is the only one that enforces CORS.
*   **Configuration**:
    *   **Development**: `AllowAllOrigins: true` is fine for convenience.
    *   **Production**: Explicitly list allowed origins. **Never** `AllowAllOrigins: true` in production.
    *   Allowed headers: Include the custom headers your API uses (`X-API-Key`, `Idempotency-Key`, etc.) in addition to standard ones (`Content-Type`, `Authorization`).
*   **Preflight requests**: The browser sends an `OPTIONS` before requests with custom headers or non-simple methods. The middleware must respond correctly or the actual request will never execute.

#### 7. Body Size Limit — MANDATORY

*   **What it does**: Limits the maximum size of the request body (e.g., 1 MB).
*   **Why it's needed**: Without this limit, an attacker (or a buggy client) can send a 10 GB body and exhaust the server's memory. It's a trivial denial of service (DoS) attack vector.
*   **Recommended value**: Depends on the use case. For typical JSON APIs, 1 MB is sufficient. For file uploads, adjust to the expected maximum and use streaming.
*   **Behavior when exceeded**: Return `413 Payload Too Large` with a clear message.

#### 8. Content-Type Enforcement — MANDATORY

*   **What it does**: Rejects requests that don't send the expected `Content-Type` (usually `application/json` for REST APIs).
*   **Why it's needed**: Prevents the server from processing unexpected payloads. An HTML form POSTing to your API would send `application/x-www-form-urlencoded` — without this validation, you might try to parse garbage as JSON and expose internal errors.
*   **Where to apply**: Only on methods that have a body (`POST`, `PUT`, `PATCH`). Don't apply on `GET` or `DELETE`.
*   **Behavior on failure**: Return `415 Unsupported Media Type`.

#### 9. Rate Limiter — MANDATORY

*   **What it does**: Limits the number of requests per client within a time period.
*   **Why it's needed**: Protects against abuse, aggressive scraping, brute force attacks, and buggy clients making infinite retry loops.
*   **Types of rate limiting**:

    | Type | How It Works | When to Use It |
    |------|-------------|----------------|
    | **Token Bucket** | Allows bursts up to a limit, then throttles to a constant rate | General case — balances flexibility and protection |
    | **Fixed Window** | Counts requests in fixed windows (e.g., 100 req/minute) | Simple to implement, but allows bursts at the window edge |
    | **Sliding Window** | Window that moves continuously | More accurate than fixed window, avoids the edge problem |
    | **Leaky Bucket** | Processes requests at a constant rate, queues the excess | When you need a strictly constant output rate |

*   **How to identify the client**: Usually by IP, API Key, or user (JWT claim). The choice depends on the authentication type.
*   **Behavior when exceeded**: Return `429 Too Many Requests` with the `Retry-After` header indicating how many seconds to wait.
*   **Consideration**: In multi-instance architectures, an in-memory rate limiter only protects per instance. For global rate limiting, use a shared store (Redis).

#### 10. Request Timeout — MANDATORY

*   **What it does**: Sets a maximum time to process each request. If exceeded, the operation is canceled.
*   **Why it's needed**: Without a timeout, a handler waiting for a response from an external service that never arrives keeps the connection open indefinitely, consuming server resources until exhaustion.
*   **Where to configure**: At the `http.Server` level (Go: `ReadTimeout`, `WriteTimeout`, `IdleTimeout`) and/or as middleware with `context.WithTimeout`.
*   **Recommended values**:

    | Timeout | Typical Value | What It Protects |
    |---------|--------------|------------------|
    | `ReadHeaderTimeout` | 5s | Slow headers attack (client sends headers slowly) |
    | `ReadTimeout` | 15s | Complete request reading (headers + body) |
    | `WriteTimeout` | 15s | Time to generate and send the response |
    | `IdleTimeout` | 60s | Inactive keep-alive connections |

*   **Propagation**: Pass the request `context` to all downstream operations (DB queries, external HTTP calls). This way, when the request timeout fires, everything cancels in cascade.

#### 11. Authentication — MANDATORY (on protected routes)

*   **What it does**: Validates the caller's identity (JWT, API Key, OAuth, etc.).
*   **Where to apply**: Only on routes that require authentication. Endpoints like `/health`, `/metrics`, and public documentation must NOT require auth.
*   **Details**: See [Section 5.4 — Authentication and Authorization](#54-authentication-and-authorization).

#### 12. Compression (gzip) — RECOMMENDED

*   **What it does**: Compresses responses before sending them to the client, reducing transfer size.
*   **When it's needed**: When responses are large (data lists, reports, exports). For small responses (< 1 KB), compression can be counterproductive (CPU overhead without real size savings).
*   **When NOT to use it**: If there's a reverse proxy in front (nginx, Cloudflare, AWS ALB) that already compresses responses. Compressing twice doesn't reduce size further and wastes CPU.
*   **Configuration**: Only compress if the client supports it (header `Accept-Encoding: gzip`). Set a minimum response size to activate compression (e.g., only if > 1 KB).

#### 13. Request Tracing (OpenTelemetry) — RECOMMENDED

*   **What it does**: Creates a span for each HTTP request and propagates the trace context to downstream operations.
*   **When it's needed**: When you have more than one service (microservices). For a simple monolith it may be overkill.
*   **Details**: See [Section 9.3 — Distributed Tracing](#93-distributed-tracing).

### 16.2 Recommended Middleware Stack Order

Order matters. This is the recommended order for the middleware chain:

```
Incoming Request
    │
    ▼
1.  Recovery / Panic Handler     ← Catches any panic from subsequent middlewares and handlers
    │
    ▼
2.  Request ID                   ← Generates/extracts ID for correlation in all logs
    │
    ▼
3.  Logger                       ← Logs request + response (with request ID already available)
    │
    ▼
4.  Security Headers             ← Adds protection headers to every response
    │
    ▼
5.  Metrics (Prometheus)         ← Measures latency and status of every request
    │
    ▼
6.  CORS                         ← Responds to preflight OPTIONS before any other validation
    │
    ▼
7.  Body Size Limit              ← Rejects large payloads before reading the body
    │
    ▼
8.  Content-Type Enforcement     ← Rejects unsupported content-types before attempting to parse
    │
    ▼
9.  Rate Limiter                 ← Limits requests before executing expensive logic
    │
    ▼
10. Authentication               ← Validates identity (JWT / API Key)
    │
    ▼
11. Handler (business logic)
```

**Why this order?**

*   **Recovery first**: If any subsequent middleware panics, recovery catches it.
*   **Request ID and Logger before everything else**: Any error (rate limit, auth failure, etc.) already has a request ID in the log.
*   **Security headers and metrics before CORS**: Every response (including preflight) gets security headers and is measured.
*   **CORS before body limit and rate limiter**: OPTIONS preflight requests have no body and shouldn't consume rate limit.
*   **Rate limiter before auth**: Limiting first prevents an attacker from brute-forcing the authentication endpoint without restriction.
*   **Auth last**: Only executes if the request already passed all previous validations.

### 16.3 Input Validation

All data entering the system from the outside is potentially malicious or incorrect. Validation is applied at different points depending on the input type.

#### Validation Types

| Type | Where to Validate | Example | What to Check |
|------|-------------------|---------|---------------|
| **Path params** | Handler / Router | `GET /users/:id` | Valid format (UUID, integer), not empty |
| **Query params** | Handler | `?page=1&limit=50` | Correct type, valid range (limit ≤ 100), allowed values for enums |
| **Headers** | Middleware or Handler | `Content-Type`, `Idempotency-Key` | Presence, expected format |
| **Body (JSON)** | Application layer | `{"email": "user@example.com"}` | Required fields, types, formats, ranges, business rules |
| **Files** | Handler | Image upload | Max size, real MIME type (don't trust the extension), dimensions |

#### What to Validate Where (by Layer)

```
HTTP Handler (infrastructure)
├── Headers correct? (Content-Type, Idempotency-Key)
├── Path params valid? (well-formed UUID)
├── Query params in range? (page > 0, limit ≤ 100)
└── JSON parseable? (valid syntax)

Application Layer (use case / validation)
├── Required fields present? (email not empty)
├── Correct formats? (email contains @, phone is numeric)
├── Values in range? (amount > 0, age between 0 and 150)
└── Business rules? (can't cancel an already-delivered order)

Domain Layer (entity)
├── Domain invariants? (valid event type, idempotency key not empty)
└── Reject invalid states at construction (constructor validates)
```

#### Validation Principles

*   **Fail fast**: Validate as early as possible. Don't execute DB queries if the input is already known to be invalid.
*   **Useful error messages**: Indicate which field failed and why. `"email: must contain @"` is useful. `"invalid request"` is not.
*   **Always validate on the backend**: Even if the frontend validates, the backend **must** validate again. The frontend is a convenience for the user; the backend is the source of truth.
*   **Don't trust the declared MIME type**: For uploads, verify the actual file content (magic bytes), not just the `Content-Type` header.
*   **Sanitize for the usage context**: There's no "universal sanitize". For SQL use parameterized queries, for HTML escape characters, for URLs encode. Each context has its own strategy.

### 16.4 Graceful Shutdown

*   **What it is**: When the server receives a shutdown signal (SIGTERM, SIGINT), it stops accepting new connections but waits for in-flight requests to finish before closing.
*   **Why it's mandatory**: Without graceful shutdown, in-flight requests are abruptly cut → 502 responses for clients → possible data corruption if a DB write is interrupted mid-way.
*   **How it works**:
    1.  The process receives `SIGTERM` (signal Kubernetes sends before killing a pod)
    2.  The server stops accepting new connections
    3.  Waits up to a timeout (e.g., 15s) for active requests to complete
    4.  Closes DB connections, flushes logs, releases resources
    5.  The process exits with code 0
*   **Shutdown timeout**: Must be **less than** Kubernetes' `terminationGracePeriodSeconds` (default 30s). If your shutdown timeout is 15s and K8s gives 30s, there's margin. If it's higher, K8s kills the process with SIGKILL.

### 16.5 TLS / HTTPS

*   **In production**: Traffic **must** be encrypted. No exceptions.
*   **Where to terminate TLS**:

    | Option | How It Works | When to Use It |
    |--------|-------------|----------------|
    | **At the load balancer / reverse proxy** | Nginx, AWS ALB, or Cloudflare terminate TLS. The backend receives plain HTTP | Most common. Simplifies the backend, centralizes certificates |
    | **At the application directly** | The backend serves HTTPS with its own cert and key | When there's no proxy in front, or for compliance requirements (end-to-end encryption) |
    | **mTLS (mutual TLS)** | Both sides present a certificate. Server validates client and vice versa | Service-to-service communication in high-security environments |

*   **Certificates**: Use Let's Encrypt (free) or the certificate your cloud provider offers. Automatic renewal.
*   If TLS is terminated at the proxy, ensure proxy → backend communication is on a private network (VPC, Kubernetes internal network).

### 16.6 Production Checklist

Quick summary of everything a backend API must have before going to production:

| Category | Requirement | Mandatory | Section |
|----------|-------------|:---------:|---------|
| **Middleware** | Recovery / Panic handler | Yes | 16.1.1 |
| **Middleware** | Request ID | Yes | 16.1.2 |
| **Middleware** | Per-request logger | Yes | 16.1.3 |
| **Middleware** | Security headers | Yes | 16.1.4 |
| **Middleware** | Prometheus metrics | Yes | 16.1.5 |
| **Middleware** | CORS | Only if frontend on different domain | 16.1.6 |
| **Middleware** | Body size limit | Yes | 16.1.7 |
| **Middleware** | Content-Type enforcement | Yes | 16.1.8 |
| **Middleware** | Rate limiting | Yes | 16.1.9 |
| **Middleware** | Request timeout | Yes | 16.1.10 |
| **Middleware** | Authentication | Yes (on protected routes) | 16.1.11 |
| **Middleware** | Gzip compression | If no proxy handles it | 16.1.12 |
| **Middleware** | OpenTelemetry tracing | If microservices | 16.1.13 |
| **Validation** | Input at all layers | Yes | 16.3 |
| **Infra** | Graceful shutdown | Yes | 16.4 |
| **Infra** | TLS / HTTPS | Yes | 16.5 |
| **Infra** | Health checks (liveness + readiness) | Yes | 13 |
| **Infra** | Configuration validated at startup | Yes | 4.5 |
| **Observability** | Structured JSON logs | Yes | 9.1 |
| **Observability** | RED metrics | Yes | 9.2 |
| **Observability** | Alerts with runbooks | Yes | 9.4 |
