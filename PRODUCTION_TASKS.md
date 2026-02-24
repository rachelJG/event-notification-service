# Production Readiness Tasks

Tareas para llevar el Event Notification Service a producción.
Canal de notificaciones: **solo email** (no webhooks ni SMS por ahora).

Estado actual: La API recibe eventos y los persiste, pero **no existe el worker asíncrono ni el envío de notificaciones**. El README describe un flujo `POST → Persist → Async Worker → Channel Delivery` que aún no está implementado.

---

## Fase 1 — Modelo de datos para notificaciones

### TASK-1.1: Migración — tabla `notifications`

Crear `002_create_notifications.up.sql` / `.down.sql`:

```sql
CREATE TYPE notification_status AS ENUM ('pending', 'processing', 'delivered', 'failed');
CREATE TYPE notification_channel AS ENUM ('email');

CREATE TABLE IF NOT EXISTS notifications (
  id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  event_id      UUID NOT NULL REFERENCES events(id),
  channel       notification_channel NOT NULL,
  recipient     TEXT NOT NULL,            -- email address
  status        notification_status NOT NULL DEFAULT 'pending',
  attempts      INT NOT NULL DEFAULT 0,
  last_error    TEXT,
  next_retry_at TIMESTAMPTZ,
  created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_notifications_pending ON notifications (status, next_retry_at)
  WHERE status IN ('pending', 'failed');
CREATE INDEX idx_notifications_event_id ON notifications (event_id);
```

**Archivos:** `internal/infrastructure/postgres/migrations/`

### TASK-1.2: Migración — columna `status` en `events`

Agregar estado al evento para saber si ya fue procesado por el worker:

```sql
ALTER TABLE events ADD COLUMN status TEXT NOT NULL DEFAULT 'accepted'
  CHECK (status IN ('accepted', 'processing', 'notified', 'failed'));
CREATE INDEX idx_events_status ON events (status) WHERE status = 'accepted';
```

**Archivos:** `internal/infrastructure/postgres/migrations/003_add_event_status.up.sql`

---

## Fase 2 — Dominio y puertos para notificaciones

### TASK-2.1: Entidad `Notification` en el dominio

Crear `internal/domain/entities/notification.go` con la entidad y value objects (`NotificationStatus`, `Channel`). Sin dependencias externas, igual que `Event`.

### TASK-2.2: Output port `NotificationRepository`

Crear `internal/domain/ports/notification_repository.go`:

```go
type NotificationRepository interface {
    Create(ctx context.Context, n entities.Notification) (string, error)
    FindPending(ctx context.Context, limit int) ([]entities.Notification, error)
    UpdateStatus(ctx context.Context, id string, status entities.NotificationStatus, lastError string) error
}
```

### TASK-2.3: Output port `EmailSender`

Crear `internal/domain/ports/email_sender.go`:

```go
type EmailSender interface {
    Send(ctx context.Context, to, subject, body string) error
}
```

Este puerto lo implementará el adaptador SMTP en la infraestructura.

### TASK-2.4: Actualizar `EventRepository` con métodos para el worker

Agregar a la interfaz:

```go
ClaimPending(ctx context.Context, limit int) ([]entities.Event, error)
SetStatus(ctx context.Context, id string, status string) error
```

`ClaimPending` debe usar `SELECT ... FOR UPDATE SKIP LOCKED` para que múltiples workers no procesen el mismo evento.

---

## Fase 3 — Lógica de aplicación (use cases)

### TASK-3.1: Use case `ProcessEvents`

Crear `internal/application/usecases/process_events.go`:
1. Llamar `EventRepository.ClaimPending(limit)`.
2. Para cada evento, resolver el destinatario según `event_type` y payload (el email viene en el payload de cada tipo de evento).
3. Crear una `Notification` (status=pending, channel=email).
4. Persistir con `NotificationRepository.Create`.
5. Marcar el evento como `processing`.

### TASK-3.2: Use case `DeliverNotifications`

Crear `internal/application/usecases/deliver_notifications.go`:
1. Llamar `NotificationRepository.FindPending(limit)`.
2. Para cada notificación, llamar `EmailSender.Send`.
3. Si éxito → status `delivered`.
4. Si error → incrementar `attempts`, poner `last_error`, calcular `next_retry_at` con backoff exponencial, y si `attempts >= maxRetries` → status `failed`.

### TASK-3.3: Input port `ProcessEventsUseCase` y `DeliverNotificationsUseCase`

Crear las interfaces en `internal/application/ports/`:

```go
type ProcessEventsUseCase interface {
    Handle(ctx context.Context) (int, error) // retorna count procesados
}

type DeliverNotificationsUseCase interface {
    Handle(ctx context.Context) (int, error)
}
```

---

## Fase 4 — Infraestructura de email

### TASK-4.1: Adaptador SMTP (`EmailSender`)

Crear `internal/infrastructure/email/smtp.go` que implemente `ports.EmailSender` usando `net/smtp` o una librería como `gomail`:
- Configuración: `SMTP_HOST`, `SMTP_PORT`, `SMTP_USER`, `SMTP_PASSWORD`, `SMTP_FROM`.
- TLS por defecto.
- Timeout configurable.

**Archivos:** `internal/infrastructure/email/smtp.go`, `internal/config/config.go`

### TASK-4.2: Templates de email por tipo de evento

Crear `internal/infrastructure/email/templates.go` con funciones que retornen subject + body HTML/texto según el `event_type`:
- `UserRegistered` → "Bienvenido, {name}"
- `PasswordResetRequested` → "Restablece tu contraseña"
- `OrderPaid` → "Confirmación de pago #{order_id}"
- `OrderShipped` → "Tu pedido #{order_id} ha sido enviado"

### TASK-4.3: Configuración SMTP en `config.go`

Agregar env vars al `Config` struct y `Load()`:
- `SMTP_HOST`, `SMTP_PORT` (default 587), `SMTP_USER`, `SMTP_PASSWORD`, `SMTP_FROM`
- Validar en `Validate()`: si `APP_ENV=production`, SMTP_HOST es requerido.

---

## Fase 5 — Worker asíncrono

### TASK-5.1: Comando `cmd/worker/main.go`

Crear un segundo binario que:
1. Carga config, logger, pool de DB.
2. Ejecuta dos loops en goroutines separadas:
   - **Process loop:** cada N segundos llama `ProcessEventsUseCase.Handle`.
   - **Deliver loop:** cada M segundos llama `DeliverNotificationsUseCase.Handle`.
3. Graceful shutdown con señales SIGINT/SIGTERM.
4. Métricas Prometheus en un puerto separado (ej. `:9090`).

**Config env vars:** `WORKER_PROCESS_INTERVAL` (default 5s), `WORKER_DELIVER_INTERVAL` (default 3s), `WORKER_BATCH_SIZE` (default 50), `WORKER_MAX_RETRIES` (default 5).

### TASK-5.2: Implementar `NotificationRepository` en Postgres

Crear `internal/infrastructure/postgres/notification_repository.go` con las queries para `Create`, `FindPending`, `UpdateStatus`.

### TASK-5.3: Implementar `ClaimPending` y `SetStatus` en `EventRepository`

Agregar a `internal/infrastructure/postgres/event_repository.go`:

```go
func (r EventRepository) ClaimPending(ctx context.Context, limit int) ([]entities.Event, error) {
    // SELECT ... FROM events WHERE status = 'accepted'
    // ORDER BY created_at FOR UPDATE SKIP LOCKED LIMIT $1
}
```

### TASK-5.4: Agregar worker al `Dockerfile` y `docker-compose.yml`

- Dockerfile multi-target: `--target api` y `--target worker`, o un segundo Dockerfile.
- docker-compose: agregar servicio `worker` con las mismas env vars de DB + las de SMTP.

### TASK-5.5: Makefile — agregar targets para el worker

```makefile
build-worker:
    go build -o bin/worker ./cmd/worker
run-worker:
    go run ./cmd/worker
```

---

## Fase 6 — Hardening para producción

### TASK-6.1: Rate limiter — detectar IP real detrás de proxy

`clientIP()` en `middleware/rate_limit.go` usa `r.RemoteAddr` directamente. En Kubernetes/load balancer todo viene de la misma IP privada.

Solución: respetar `X-Forwarded-For` con un `TRUSTED_PROXY_COUNT` configurable.

**Archivos:** `internal/infrastructure/http/middleware/rate_limit.go`, `internal/config/config.go`

### TASK-6.2: Header `Retry-After` en respuestas 429

Cuando el rate limiter rechaza, incluir `Retry-After: <seconds>` para que los clientes hagan backoff.

**Archivos:** `internal/infrastructure/http/middleware/rate_limit.go`

### TASK-6.3: Devolver el ID original en respuesta 409 Conflict

El INSERT ON CONFLICT ya recupera el ID original. Propagarlo en el body del 409 para que el cliente no necesite un GET adicional.

**Archivos:** `internal/infrastructure/http/handler.go`, `internal/infrastructure/http/errmap/`

### TASK-6.4: Header `Cache-Control: no-store` en rutas API

Prevenir que proxies o navegadores cacheen datos de eventos.

**Archivos:** `internal/infrastructure/http/server.go` (middleware en el grupo `/api/v1`)

### TASK-6.5: Validar valores positivos en `config.Validate()`

Agregar checks: `RateLimitRPS > 0`, `RateLimitBurst > 0`, `BodyLimitBytes > 0`, `DBQueryTimeout > 0`.

**Archivos:** `internal/config/config.go`

### TASK-6.6: Shutdown timeout configurable

Agregar `SHUTDOWN_TIMEOUT` env var (default 15s) en lugar de hard-coded en `cmd/api/main.go`.

**Archivos:** `cmd/api/main.go`, `internal/config/config.go`

---

## Fase 7 — Observabilidad

### TASK-7.1: Métricas de pgxpool en Prometheus

Implementar `prometheus.Collector` que exponga stats del pool de conexiones (total, idle, acquired, wait count).

**Archivos:** nuevo `internal/infrastructure/postgres/metrics.go`, `cmd/api/main.go`

### TASK-7.2: Contador de errores por código

Agregar `http_errors_total{code}` para alertar sobre spikes de errores por tipo.

**Archivos:** `internal/infrastructure/http/handler.go` o `errmap/`

### TASK-7.3: Métricas del worker

Contadores: `events_processed_total{result}`, `notifications_delivered_total{channel,result}`, `notification_retries_total`. Gauge: `pending_notifications`.

**Archivos:** `cmd/worker/`, `internal/application/usecases/`

### TASK-7.4: Audit logging en JWT middleware

Loguear autenticación exitosa (sub claim) y fallida (razón + IP remota) con campo estructurado `event: "auth"`.

**Archivos:** `internal/infrastructure/http/middleware/jwt.go`

---

## Fase 8 — Testing

### TASK-8.1: Tests unitarios para `ProcessEvents` y `DeliverNotifications`

Usar fakes de `NotificationRepository`, `EventRepository`, `EmailSender`. Cubrir: éxito, error de email, max retries, backoff.

### TASK-8.2: Tests unitarios para el adaptador SMTP

Levantar un servidor SMTP fake in-process. Verificar que el email se construye correctamente.

### TASK-8.3: Tests de health endpoints

`/health/live` y `/health/ready` no tienen cobertura. Agregar tests con fake `HealthChecker`.

**Archivos:** `internal/infrastructure/http/handler_test.go`

### TASK-8.4: Test de integración del worker

Con Postgres real: insertar eventos, correr `ProcessEvents`, verificar que se crean notificaciones. Correr `DeliverNotifications` con un `EmailSender` fake, verificar status `delivered`.

**Archivos:** `internal/tests/`

### TASK-8.5: Subir threshold de coverage a 60%

Editar `.github/workflows/ci.yml` de 30% a 60% después de agregar los tests anteriores.

---

## Fase 9 — Despliegue

### TASK-9.1: Kubernetes manifests

Crear `deploy/k8s/` con:
- `deployment-api.yaml` — probes, resources, envFrom Secret/ConfigMap
- `deployment-worker.yaml` — igual pero sin probes HTTP (usa exec probe)
- `service.yaml` — ClusterIP para la API
- `configmap.yaml` — config no secreta
- `secret.yaml` (template) — JWT_SECRET, PG_DSN, SMTP_PASSWORD

### TASK-9.2: Completar deploy workflow en CI

Reemplazar el placeholder en `.github/workflows/deploy.yml` con pasos reales (build image → push registry → kubectl apply / helm upgrade).

### TASK-9.3: Build del worker en CI

Agregar step en `ci.yml` para compilar y testear el binario del worker.

---

## Resumen de prioridades

| Fase | Descripción | Dependencias |
|------|------------|--------------|
| 1 | Modelo de datos (migraciones) | Ninguna |
| 2 | Dominio y puertos | Fase 1 |
| 3 | Use cases (lógica de negocio) | Fase 2 |
| 4 | Adaptador SMTP | Fase 2 |
| 5 | Worker asíncrono | Fases 3 + 4 |
| 6 | Hardening de la API existente | Independiente, se puede hacer en paralelo |
| 7 | Observabilidad | Parcialmente independiente, métricas del worker dependen de Fase 5 |
| 8 | Testing | Después de cada fase |
| 9 | Despliegue | Después de Fase 5 |
