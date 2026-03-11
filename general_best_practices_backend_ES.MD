# Mejores Prácticas para Desarrollo Backend

> **Última actualización:** 2026-03-11
>
> Esta guía establece estándares de calidad, mantenibilidad y escalabilidad para proyectos backend. Para frontend existe un documento separado.

---

## 1. Sistema de Control de Versiones (Commits y Ramas)

*   **Conventional Commits**: Obligatorio para historiales legibles y automatización (CHANGELOG y SemVer).
    *   `feat:`: Nueva funcionalidad. Ej: `feat: añade endpoint de facturación`.
    *   `fix:`: Corrección de error. Ej: `fix: corrige timeout en consulta de pagos`.
    *   `docs:`: Actualiza documentación. Ej: `docs: actualiza readme`.
    *   `chore:`, `refactor:`, `test:`, `ci:`, `perf:` para otras tareas.
    *   `BREAKING CHANGE:`: Cambio que rompe compatibilidad hacia atrás (incrementa versión mayor).
*   **Versionado Semántico (SemVer)**: La versión en git debe seguir `MAJOR.MINOR.PATCH`. Herramientas como `semantic-release` o `release-please` automatizan esto a partir de los commits.
*   **Commits Atómicos**: Un commit = un cambio lógico. El código debe compilar y los tests pasar en cada commit del historial.
*   **Estrategia de Ramas (GitHub Flow / Trunk-based)**:
    *   La rama `main` siempre está protegida y desplegable.
    *   Nadie hace push directo a `main`. Todo entra por Pull Requests (PR) desde ramas de vida corta.
*   **Estrategia de Merge**: Elegir según el equipo y mantener consistencia:
    *   **Squash and Merge**: Condensa todos los commits del PR en uno solo. Ideal cuando los commits intermedios son ruidosos (WIP, fixups). Historial limpio en `main`.
    *   **Rebase and Merge**: Preserva cada commit individual. Preferible cuando los commits ya son atómicos y descriptivos. Mantiene un historial lineal y detallado.
    *   Lo importante es que el equipo elija una estrategia y la aplique de forma consistente.

---

## 2. Code Review

El code review es una práctica obligatoria. Todo cambio pasa por revisión antes de llegar a `main`.

*   **Mínimo 1 aprobación** antes de hacer merge. En sistemas críticos, considerar 2.
*   **CODEOWNERS**: Configurar `.github/CODEOWNERS` para asignar revisores automáticos por área del código (ej: `internal/infrastructure/postgres/ @equipo-data`).
*   **Qué revisar** (en orden de prioridad):
    1.  **Correctitud**: ¿Hace lo que dice que hace? ¿Maneja errores correctamente?
    2.  **Seguridad**: ¿Introduce vulnerabilidades? ¿Expone datos sensibles?
    3.  **Diseño**: ¿Respeta la arquitectura? ¿Las abstracciones son adecuadas?
    4.  **Legibilidad**: ¿Se entiende sin necesidad de explicación verbal?
    5.  **Rendimiento**: ¿Hay consultas N+1, allocations innecesarias, o bloqueos?
*   **Buenas prácticas del revisor**:
    *   Comentarios constructivos: explicar el "por qué", no solo el "qué".
    *   Distinguir entre bloqueos (must fix) y sugerencias (nice to have).
    *   No bloquear PRs por estilo si hay un linter que lo maneja.
*   **Buenas prácticas del autor**:
    *   PRs pequeños y enfocados (< 400 líneas idealmente).
    *   Descripción clara del PR: qué cambia, por qué, cómo probarlo.
    *   Auto-revisar el diff antes de solicitar revisión.

---

## 3. Estructura del README

El `README.md` es la presentación del proyecto. Estructura recomendada:

1.  **Título e Insignias (Badges)**: CI status, coverage, versión.
2.  **Descripción**: Qué hace el proyecto y por qué existe.
3.  **Vista General (Overview)**: Características principales, stack tecnológico y diagrama de arquitectura.
4.  **Prerrequisitos**: Dependencias necesarias (ej: Go 1.22, Docker, PostgreSQL 15).
5.  **Instalación y Configuración**: Pasos concretos (ej: copiar `.env.example` a `.env`).
6.  **Ejecución Local**: Comandos para iniciar la app (ej: `make run`, `docker-compose up`).
7.  **Estructura del Proyecto**: Breve explicación de carpetas clave.
8.  **Ejecución de Pruebas**: Comandos para correr tests y linters.
9.  **Uso / Endpoints**: Ejemplos `curl` o enlace a OpenAPI/Swagger.

---

## 4. Gestión de Configuración

La configuración de una aplicación es todo aquello que varía entre entornos (desarrollo, staging, producción): credenciales de DB, puertos, URLs de servicios externos, feature flags, etc. Seguir el principio de [Twelve-Factor App: Config](https://12factor.net/config).

### 4.1 Regla de oro: Configuración fuera del código

*   La configuración se inyecta mediante **variables de entorno**, nunca hardcodeada en el código fuente.
*   Si un valor cambia entre entornos, es configuración. Si es igual en todos los entornos, es una constante del código.

### 4.2 Archivos `.env` — Solo para desarrollo local

*   El archivo `.env` es una conveniencia para desarrollo local. **Nunca** debe existir en servidores de producción.
*   **`.env`** → Contiene los valores reales para tu máquina local. **Nunca comitearlo** (debe estar en `.gitignore`).
*   **`.env.example`** → Sí se commitea al repositorio. Sirve como documentación viva de todas las variables que la aplicación necesita. Contiene valores de ejemplo o vacíos, nunca secretos reales.
    ```env
    # .env.example
    API_ADDR=:8080
    PG_DSN=postgres://postgres:postgres@localhost:5432/mydb?sslmode=disable
    LOG_LEVEL=info
    APP_ENV=development

    # Secretos (dejar vacíos, cada dev pone los suyos)
    SMTP_PASSWORD=
    API_KEY_SECRET=
    ```
*   Al clonar el proyecto, el primer paso es: `cp .env.example .env` y rellenar los valores faltantes.

### 4.3 Configuración en producción

En producción, las variables de entorno se inyectan desde el orquestador o la plataforma, **no desde archivos `.env`**:

| Plataforma | Cómo se configuran las variables |
|------------|----------------------------------|
| **Kubernetes** | `ConfigMap` (no sensibles) + `Secret` (sensibles), montados como env vars en el pod |
| **Docker Compose** | Sección `environment:` en el `docker-compose.yml` o archivo `env_file:` |
| **AWS ECS/Fargate** | Task Definition → environment + Secrets Manager references |
| **Railway / Render / Fly.io** | Dashboard de la plataforma → Variables de entorno |
| **Systemd** | `Environment=` o `EnvironmentFile=` en el unit file |

### 4.4 Separar configuración de secretos

No todo es igual de sensible. Distinguir entre:

*   **Configuración no sensible**: Puertos, niveles de log, timeouts, tamaños de pool, feature flags. Pueden vivir en `ConfigMap`, archivos de configuración, o variables de entorno sin protección adicional.
*   **Secretos**: Contraseñas de DB, API keys, tokens, certificados TLS. Deben almacenarse en sistemas dedicados:
    *   Kubernetes Secrets (encriptados at rest con KMS)
    *   HashiCorp Vault
    *   AWS Secrets Manager / GCP Secret Manager / Azure Key Vault
*   **Nunca** loguear secretos. Sanitizar las variables sensibles antes de imprimirlas en logs de arranque.

### 4.5 Validación al arrancar

*   Validar **toda** la configuración al iniciar la aplicación. Si falta una variable requerida o tiene un valor inválido, la app debe **fallar inmediatamente** con un mensaje claro.
    ```
    invalid config: PG_DSN is required
    invalid config: RATE_LIMIT_RPS must be > 0
    ```
*   No esperar a que la primera request falle para descubrir que falta configuración.
*   Validar coherencia: si `HSTS_ENABLED=true`, entonces `HSTS_MAX_AGE_SECONDS` debe ser > 0. Si `TLS_CERT_FILE` está definido, `TLS_KEY_FILE` también debe estarlo.
*   Proveer **defaults sensatos** para desarrollo: la app debe arrancar con `cp .env.example .env` sin modificar nada para el caso más básico.

### 4.6 Estructura recomendada en código

```
// Centralizar en un solo paquete (ej: internal/config/)
config.Load()      → Lee todas las variables de entorno con defaults
config.Validate()  → Valida coherencia y campos requeridos
```

*   Un solo lugar donde se lee la configuración, no `os.Getenv()` disperso por todo el código.
*   Las capas internas reciben solo los valores que necesitan (structs específicos o campos individuales), no el objeto `Config` completo. Esto respeta la inversión de dependencias.

---

## 5. Seguridad

La seguridad no es opcional. Debe estar presente desde el diseño, no añadida al final.

### 5.1 Gestión de Secretos
*   **Nunca** hardcodear credenciales, tokens o claves en el código fuente (ver [Sección 4.4](#44-separar-configuración-de-secretos)).
*   Para entornos con múltiples secretos, usar un gestor dedicado (HashiCorp Vault, AWS Secrets Manager, GCP Secret Manager).
*   Agregar `.env` al `.gitignore` (ver [Sección 4.2](#42-archivos-env--solo-para-desarrollo-local)).

### 5.2 Dependencias
*   Mantener un **lock file** (`go.sum`, `package-lock.json`, `poetry.lock`) y comitearlo al repositorio.
*   Ejecutar escaneos de vulnerabilidades de forma periódica y manual: `govulncheck ./...` (Go), `npm audit` (Node), `pip-audit` (Python).
*   Antes de actualizar dependencias, verificar el changelog y correr la suite de tests completa. Las actualizaciones deben ser deliberadas, no automáticas.
*   Antes de agregar una dependencia nueva, evaluar: ¿está activamente mantenida? ¿Cuál es su licencia? ¿Cuántas dependencias transitivas trae?

### 5.3 OWASP Top 10 (las más relevantes para backend)
*   **Inyección (SQL, NoSQL, OS)**: Usar siempre consultas parametrizadas o prepared statements. Nunca interpolar input del usuario en queries.
*   **Autenticación rota**: Hashear contraseñas con bcrypt/argon2 (nunca MD5/SHA1 para passwords). Implementar rate limiting en endpoints de login.
*   **Exposición de datos sensibles**: No retornar datos internos en respuestas de error (stack traces, queries SQL, rutas del filesystem).
*   **Control de acceso roto**: Validar permisos en el backend, nunca confiar en validaciones del cliente. Aplicar principio de mínimo privilegio.
*   **Configuración insegura**: Deshabilitar endpoints de debug en producción. Configurar headers de seguridad (HSTS, Content-Security-Policy, X-Content-Type-Options).

### 5.4 Autenticación y Autorización

Existen dos mecanismos principales para proteger APIs. No son intercambiables — cada uno resuelve un problema distinto.

#### JWT (JSON Web Token) — Identidad de usuarios

*   **Cuándo usarlo**: Cuando necesitas saber **quién** es el usuario (login, sesiones, permisos por usuario).
*   **Cómo funciona**: El cliente se autentica (login), recibe un token firmado que contiene claims (`sub`, `exp`, `roles`). El backend valida la firma sin consultar la DB en cada request.
*   **Header estándar**: `Authorization: Bearer <token>` (RFC 6750, estándar OAuth 2.0).
    ```
    Authorization: Bearer eyJhbGciOiJIUzI1NiIs...
    ```
*   **Buenas prácticas**:
    *   Algoritmo seguro: HS256 (secreto compartido) o RS256/ES256 (clave asimétrica).
    *   Siempre exigir `exp` (expiración). Tokens de corta vida (15-60 min) + refresh token.
    *   Validar `iss` (issuer) y `aud` (audience) si el sistema tiene múltiples servicios.
    *   Secreto mínimo de 32 bytes para HS256.
    *   **Nunca** almacenar datos sensibles en el payload — el JWT es firmado, no encriptado (cualquiera puede leer el contenido con base64).
*   **Casos típicos**: Apps con usuarios (SaaS, e-commerce, redes sociales), SPAs, mobile apps.

#### API Key — Identidad de sistemas/servicios

*   **Cuándo usarlo**: Comunicación **servicio-a-servicio**, integraciones de terceros, APIs públicas con plan de uso, webhooks.
*   **Cómo funciona**: Se genera una clave única por cliente/servicio. El backend busca el hash de la clave en la DB para validarla y determinar sus permisos (scopes).
*   **Header estándar**: `X-API-Key: <key>` (convención de la industria: AWS, Stripe, Google Cloud, Twilio).
    ```
    X-API-Key: ens_a1b2c3d4e5f6...
    ```
*   **NO usar** `Authorization: Bearer` para API keys — ese esquema es para tokens OAuth 2.0 (RFC 6750). Mezclarlos causa confusión y rompe la semántica del estándar.
*   **Buenas prácticas**:
    *   Almacenar solo el **hash SHA-256** de la clave, nunca el valor en claro.
    *   Mostrar la clave completa **una sola vez** al momento de crearla.
    *   Implementar **scopes** (permisos granulares): `events:read`, `events:write`, `admin`.
    *   Permitir **revocar** claves individualmente sin afectar a los demás clientes.
    *   Usar un prefijo identificable (ej: `ens_`, `sk_`, `pk_`) para facilitar la detección de leaks en logs y escaneos.
    *   Registrar `last_used_at` para identificar claves inactivas.
*   **Casos típicos**: APIs públicas (Stripe, Twilio), integraciones B2B, CI/CD pipelines, microservicios internos.

#### ¿Cuándo usar cuál?

| Escenario | Mecanismo | Razón |
|-----------|-----------|-------|
| App web/mobile con login de usuarios | JWT | Necesitas identidad del usuario y sesiones |
| API pública para terceros | API Key | El consumidor es un sistema, no un humano |
| Microservicio A llama a microservicio B | API Key o mTLS | Identidad de servicio, no de usuario |
| Webhook que recibe eventos externos | API Key + firma HMAC | Validar origen y autenticidad del payload |
| API interna + usuarios con roles | JWT | Permisos granulares por usuario |
| Dashboard admin accedido por personas | JWT | Sesión de usuario con roles |

En algunos sistemas se combinan ambos: **API Key** para identificar al cliente/tenant + **JWT** para identificar al usuario dentro de ese tenant.

### 5.5 Principio de Mínimo Privilegio
*   Usuarios de base de datos con solo los permisos necesarios (no usar el superusuario para la aplicación).
*   Tokens de CI/CD con scope limitado al repositorio o acción específica.
*   Service accounts con permisos granulares, no roles de administrador.

---

## 6. Pruebas (Testing Pyramid)

Los porcentajes son orientativos y deben ajustarse según el tipo de proyecto. Un servicio con poca lógica pura puede necesitar más integración que unitarias.

*   **Pruebas Unitarias (~70%)**:
    *   Prueban lógica de negocio aislada (funciones puras, validaciones, transformaciones).
    *   Uso de Mocks/Stubs para aislar dependencias externas (DB, APIs, filesystem).
    *   Deben ser rápidas (< 1s por test) y deterministas (sin dependencia de estado externo).
*   **Pruebas de Integración (~25%)**:
    *   Validan la interacción real con la base de datos, caches, colas de mensajes.
    *   **Mejor práctica**: Levantar dependencias reales en contenedores Docker efímeros (Testcontainers o docker-compose en CI).
    *   Verifican que las queries SQL, migraciones y mappings funcionan contra el motor real.
*   **Pruebas End-to-End (~5%)**:
    *   Prueban el flujo completo: HTTP request → lógica → base de datos → HTTP response.
    *   Reservar para los flujos críticos del negocio (ej: crear pedido, procesar pago).
    *   Son las más lentas y frágiles; minimizar su cantidad.
*   **Cobertura mínima**: Establecer un umbral (ej: 80%) en CI y bloquearlo. La cobertura no garantiza calidad, pero su ausencia garantiza puntos ciegos.
*   **Race detector**: En lenguajes que lo soporten (Go: `-race`), ejecutar siempre en CI para detectar condiciones de carrera.

---

## 7. Manejo de Errores

Los errores son ciudadanos de primera clase. Un mal manejo de errores es la causa más común de bugs difíciles de diagnosticar en producción.

*   **Nunca silenciar errores**: No usar `_ = doSomething()` sin justificación documentada.
*   **Errores tipados**: Definir una taxonomía de errores de la aplicación (ej: `NotFound`, `Conflict`, `InvalidArgument`, `Internal`). Esto permite que cada capa reaccione correctamente.
*   **Propagar con contexto**: Envolver errores con información adicional al propagarlos. En Go: `fmt.Errorf("crear usuario: %w", err)`. En otros lenguajes, usar excepciones con mensajes descriptivos.
*   **Separar errores de negocio de errores de infraestructura**: Un "usuario no encontrado" no es lo mismo que "la base de datos no responde". El primero es esperado; el segundo es una falla.
*   **Mapeo centralizado a HTTP**: La capa HTTP traduce errores de negocio a códigos HTTP en un solo lugar. Los handlers nunca deben elegir el status code directamente desde lógica de negocio.
*   **Reintentos con servicios externos**: Usar backoff exponencial con jitter. Definir un máximo de reintentos. Implementar circuit breaker si el servicio falla de forma sostenida.

---

## 8. Documentación de Código

*   **Código auto-documentado**: Nombres claros de variables, funciones y métodos. El código dice el "qué".
*   **Comentarios del "por qué"**: Explicar decisiones inusuales, workarounds o restricciones de negocio. No repetir lo que el código ya dice.
*   **Architecture Decision Records (ADR)**: Documentar decisiones técnicas importantes en `docs/adr/` (ej: `001-uso-de-kafka.md`, `002-migracion-a-api-keys.md`). Formato: contexto, decisión, consecuencias.
*   **Contratos de API**: Usar **OpenAPI/Swagger** obligatoriamente para documentar endpoints. Es el contrato entre backend y sus consumidores. Mantenerlo actualizado con el código.
*   **Documentar funciones y tipos exportados**: En Go, todo símbolo exportado debe tener un comentario que inicie con el nombre del símbolo. En otros lenguajes, seguir la convención del ecosistema (JSDoc, docstrings, etc.).

---

## 9. Observabilidad

Un sistema que no se puede observar no se puede operar. La observabilidad se basa en tres pilares:

### 9.1 Logs
*   Formato **JSON estructurado** a `stdout`/`stderr` (nunca escribir a archivos directamente).
*   Incluir siempre: `timestamp`, `level`, `message`, `request_id`/`trace_id`.
*   Niveles con propósito: `DEBUG` para desarrollo, `INFO` para flujo normal, `WARN` para situaciones recuperables, `ERROR` para fallos que requieren atención.
*   No loguear datos sensibles (tokens, contraseñas, datos personales).

### 9.2 Métricas
*   Exponer métricas en formato **Prometheus** (`/metrics`).
*   Métricas RED para cada servicio: **Rate** (requests/s), **Errors** (tasa de errores), **Duration** (latencia p50, p95, p99).
*   Métricas de negocio: eventos procesados, pagos completados, etc. Separadas de las métricas de infraestructura.
*   Métricas de recursos: pool de conexiones de DB, goroutines activas, uso de memoria.

### 9.3 Tracing Distribuido
*   Implementar **OpenTelemetry** como estándar para tracing.
*   Propagar `trace_id` en todas las llamadas entre servicios.
*   Instrumentar: handlers HTTP, queries a DB, llamadas a servicios externos.
*   El tracing permite visualizar el flujo completo de una request a través de múltiples servicios.

### 9.4 Alertas
*   Alertar sobre **síntomas** (latencia alta, tasa de errores elevada), no sobre causas (CPU al 80%).
*   Definir **SLIs** (Service Level Indicators) y **SLOs** (Service Level Objectives) para los servicios críticos.
*   Toda alerta debe tener un runbook asociado: qué significa, cómo diagnosticar, cómo remediar.
*   Evitar alert fatigue: si una alerta se ignora consistentemente, ajustarla o eliminarla.

---

## 10. Estándares de API REST

### 10.1 Diseño
*   **Versionado**: Prefijo en la URL (`/api/v1/`, `/api/v2/`). Nunca romper un contrato existente sin versión nueva.
*   **Recursos como sustantivos**: `/api/v1/users`, no `/api/v1/getUsers`.
*   **Verbos HTTP correctos**: `GET` (leer), `POST` (crear), `PUT` (reemplazar), `PATCH` (actualizar parcial), `DELETE` (eliminar).
*   **Códigos de estado consistentes**:
    *   `200` OK, `201` Created, `202` Accepted (operaciones asíncronas)
    *   `400` Bad Request, `401` Unauthorized, `403` Forbidden, `404` Not Found, `409` Conflict, `429` Too Many Requests
    *   `500` Internal Server Error, `503` Service Unavailable, `504` Gateway Timeout

### 10.2 Respuestas de Error Estructuradas
*   Formato consistente en todos los endpoints:
    ```json
    {
      "error": "descripción legible para el desarrollador",
      "code": "INVALID_ARGUMENT",
      "request_id": "abc-123"
    }
    ```
*   Nunca exponer detalles internos (stack traces, nombres de tablas, queries SQL) en las respuestas de error.

### 10.3 Paginación, Filtrado y Ordenamiento
*   **Paginación**: Usar cursor-based (`?cursor=abc&limit=20`) para conjuntos grandes, u offset-based (`?page=1&per_page=20`) para conjuntos pequeños y predecibles.
*   **Filtrado**: Query params descriptivos (`?status=active&type=payment`).
*   **Ordenamiento**: `?sort=created_at&order=desc`.
*   Documentar los valores permitidos y los defaults.

### 10.4 Idempotencia
*   Operaciones `POST` que crean recursos deben soportar un header `Idempotency-Key` para prevenir duplicados ante reintentos del cliente.
*   `PUT` y `DELETE` son idempotentes por definición.

---

## 11. Arquitectura y Estructura

*   **Arquitectura Limpia**: Para proyectos empresariales, usar una arquitectura que separe estrictamente la lógica de negocio de la infraestructura (Hexagonal, Onion, Clean Architecture). El principio clave es la **inversión de dependencias**: el dominio no depende de frameworks, bases de datos ni transporte HTTP.
*   **Estructura de carpetas por capas**:
    ```
    cmd/            → Entry points (main.go)
    internal/
      domain/       → Entidades, value objects, interfaces de puertos de salida
      application/  → Casos de uso, validación, interfaces de puertos de entrada
      infrastructure/ → Implementaciones: HTTP, DB, mensajería, servicios externos
      pkg/          → Utilidades compartidas dentro del proyecto
    ```
*   **No exponer internos**: Usar `internal/` para prevenir que otros módulos importen paquetes que no deberían.
*   **Inyección de dependencias en el composition root**: Toda la creación de objetos y wiring se hace en `cmd/*/main.go`. Las capas internas reciben sus dependencias como interfaces.

---

## 12. Integración Continua (CI)

*   **Obligatorio en cada Pull Request**:
    *   Linter (`golangci-lint`, `ruff`, `eslint`, etc.)
    *   Tests unitarios con race detector donde aplique
    *   Verificación de cobertura mínima
    *   Escaneo de vulnerabilidades en dependencias
*   **Bloquear PRs que fallen**: Configurar branch protection en `main` para requerir que todos los checks pasen.
*   **Builds reproducibles**: Fijar versiones de herramientas en CI (Go, linters, etc.). Usar lock files.
*   **Artefactos**: Guardar reportes de cobertura y binarios como artefactos del pipeline.

---

## 13. Despliegue Continuo (CD)

*   **Estrategias de despliegue**:
    *   **Rolling update**: Reemplaza instancias gradualmente. Simple y adecuado para la mayoría de servicios.
    *   **Blue-Green**: Dos ambientes idénticos; cambias el tráfico de uno a otro. Rollback instantáneo.
    *   **Canary**: Envía un porcentaje pequeño de tráfico a la nueva versión. Permite detectar problemas antes de un rollout completo.
*   **Health checks**: Todo servicio debe exponer endpoints que el orquestador use para determinar el estado de la instancia. En **Kubernetes**, hay 3 probes que se deben implementar:

    | Probe | Endpoint sugerido | Pregunta que responde | Si falla... |
    |-------|-------------------|----------------------|-------------|
    | **Liveness** | `GET /healthz` | ¿El proceso está vivo y no colgado? | K8s **reinicia** el pod |
    | **Readiness** | `GET /readyz` | ¿Puede recibir tráfico? (DB conectada, migraciones aplicadas) | K8s **deja de enviarle** requests (lo saca del Service, pero no lo reinicia) |
    | **Startup** | `GET /healthz` | ¿Ya terminó de arrancar? | K8s espera; si no pasa en el tiempo configurado, reinicia el pod |

    *   **Liveness** (`/healthz`): Debe ser ligero y rápido. Solo verifica que el proceso responde. **No** verificar dependencias externas aquí — si la DB se cae, no quieres que K8s reinicie todos los pods en cascada.
    *   **Readiness** (`/readyz`): Aquí sí verificar dependencias: conexión a DB, caches, servicios críticos. Si la DB no responde, el pod se marca como "no ready" y deja de recibir tráfico, pero sigue vivo esperando que la dependencia se recupere.
    *   **Startup**: Útil para servicios con arranque lento (precalentar caches, cargar modelos). Usa el mismo endpoint que liveness pero con un timeout más generoso (`failureThreshold * periodSeconds`).
    *   Los endpoints de health **no requieren autenticación**.
    *   Ejemplo de configuración en el deployment de Kubernetes:
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
*   **Rollback**: Debe ser posible volver a la versión anterior en menos de 5 minutos. Si no lo es, el proceso de deploy necesita mejorarse.
*   **Migraciones de base de datos**: Deben ser compatibles hacia adelante (forward-compatible). Primero deployar el schema nuevo, luego el código que lo usa. Nunca hacer cambios destructivos (DROP COLUMN) en el mismo deploy que cambia el código.
*   **Feature Flags**: Usar feature flags para desacoplar el deploy del release. Permite activar/desactivar funcionalidades sin re-deploy y hacer rollback instantáneo de features específicas.

---

## 14. Gestión de Dependencias

*   **Lock files obligatorios**: Comitear siempre `go.sum`, `package-lock.json`, `poetry.lock`, etc. Garantiza builds reproducibles.
*   **Actualizaciones deliberadas**: Las dependencias se actualizan de forma manual y consciente, no automática. Proceso recomendado:
    1.  Revisar el changelog de la nueva versión
    2.  Verificar si hay breaking changes
    3.  Actualizar en una rama dedicada
    4.  Correr la suite completa de tests
    5.  Revisar y hacer merge como cualquier otro PR
*   **Evaluar antes de agregar**: Antes de añadir una dependencia nueva, preguntarse:
    *   ¿Está activamente mantenida? (último commit, issues abiertos)
    *   ¿Cuál es su licencia? (MIT, Apache 2.0 son seguras; GPL puede ser problemática)
    *   ¿Cuántas dependencias transitivas trae?
    *   ¿Se puede resolver con código propio en menos de 50 líneas?
*   **Escaneo de vulnerabilidades**: Ejecutar periódicamente `govulncheck ./...` (Go), `npm audit` (Node) u equivalente del ecosistema. Integrar en CI como check no bloqueante inicialmente, bloqueante cuando el equipo esté listo.

---

## 15. Base de Datos

*   **Migraciones versionadas**: Usar archivos SQL numerados (`001_create_users.up.sql` / `001_create_users.down.sql`). Cada migración corre dentro de una transacción.
*   **Siempre tener down migration**: Toda migración `up` debe tener su correspondiente `down` para poder revertir.
*   **Nunca modificar migraciones ya aplicadas**: Si necesitas cambiar algo, crea una migración nueva.
*   **Connection pooling**: Configurar el pool de conexiones con límites explícitos (`max_conns`, `min_conns`, `max_conn_lifetime`, `max_conn_idle_time`). Monitorear el uso del pool.
*   **Timeouts por query**: Usar `context.WithTimeout` en cada consulta. Un query sin timeout es un riesgo de cascading failure.
*   **Índices**: Crear índices para las queries que se ejecutan frecuentemente. Usar `EXPLAIN ANALYZE` para validar que se usan correctamente.
*   **Datos sensibles**: Hashear lo que no necesitas leer (contraseñas, API keys). Encriptar lo que necesitas leer pero no en claro (datos personales, dependiendo del compliance).

---

## 16. API Lista para Producción — Middlewares y Validación

Un API que funciona en desarrollo no está lista para producción. Esta sección detalla los **middlewares mínimos** que toda API backend debe tener antes de salir a producción, los **tipos de validación** de input, y otros aspectos que a menudo se olvidan.

### 16.1 Stack de Middlewares

Los middlewares se ejecutan en orden para cada request. El **orden importa** — por ejemplo, el request ID debe asignarse antes que el logger para que aparezca en todos los logs.

A continuación se listan todos los middlewares comunes, indicando cuáles son **obligatorios** para producción y cuáles son **recomendados** según el caso.

#### 1. Recovery / Panic Handler — OBLIGATORIO

*   **Qué hace**: Captura panics no controlados dentro de un handler y retorna un `500 Internal Server Error` en lugar de crashear el proceso entero.
*   **Por qué es necesario**: Sin este middleware, un panic en un handler (nil pointer, index out of range) mata el proceso completo y afecta a todos los usuarios, no solo a esa request.
*   **Ejemplo**: Un handler accede a `user.Address.City` pero `Address` es nil → panic → sin recovery, el servidor muere.
*   **Nota**: Que exista recovery no significa que los panics estén bien. Cada panic capturado debe loguearse como `ERROR` con stack trace para investigación.

#### 2. Request ID — OBLIGATORIO

*   **Qué hace**: Asigna un identificador único (UUID v4 o similar) a cada request entrante. Lo propaga en el contexto y lo incluye en el header de respuesta (`X-Request-ID`).
*   **Por qué es necesario**: Sin un request ID, correlacionar logs entre servicios o con el cliente es imposible. Cuando un usuario reporta un error, el request ID es lo que permite encontrar la traza completa.
*   **Comportamiento**: Si el cliente ya envía un `X-Request-ID` (ej: un gateway o load balancer), reutilizarlo en lugar de generar uno nuevo. Esto mantiene la trazabilidad end-to-end.

#### 3. Logger — OBLIGATORIO

*   **Qué hace**: Loguea cada request entrante y su respuesta: método, path, status code, latencia, IP del cliente, request ID.
*   **Por qué es necesario**: Es la fuente primaria de información para debugging, auditoría y análisis de tráfico. Sin logs por request, diagnosticar problemas en producción es adivinar.
*   **Debe ejecutarse después de**: Recovery (para loguear panics) y Request ID (para incluirlo en cada log).
*   **Qué NO loguear**: Cuerpos de request/response completos (por volumen y privacidad), headers de autenticación, datos personales.

#### 4. Security Headers — OBLIGATORIO

*   **Qué hace**: Agrega headers HTTP que protegen contra ataques comunes del navegador.
*   **Por qué es necesario**: Aunque el backend no sirva HTML directamente, estas protecciones evitan que las respuestas sean mal interpretadas por el navegador.
*   **Headers mínimos**:

    | Header | Valor | Protege contra |
    |--------|-------|----------------|
    | `X-Content-Type-Options` | `nosniff` | El navegador adivine el tipo MIME y ejecute contenido malicioso |
    | `X-Frame-Options` | `DENY` o `SAMEORIGIN` | Clickjacking (embeber tu sitio en un iframe) |
    | `X-XSS-Protection` | `0` | Desactiva el filtro XSS del navegador (puede causar más problemas de los que resuelve en navegadores modernos) |
    | `Strict-Transport-Security` (HSTS) | `max-age=31536000; includeSubDomains` | Man-in-the-middle al forzar HTTPS. **Solo activar si el servicio ya usa TLS** |
    | `Content-Security-Policy` | `default-src 'none'` | Inyección de scripts. Para APIs puras que solo retornan JSON, este valor es suficiente |
    | `Referrer-Policy` | `no-referrer` | Filtración de URLs en el header Referer |

*   **¿HSTS es obligatorio?**: Solo si tu servicio ya está detrás de TLS (directamente o a través de un load balancer). Si sirves HTTP plano en desarrollo, HSTS no debe estar activo.
*   **¿Content-Security-Policy en un API?**: Sí, con `default-src 'none'`. Si alguien accede a un endpoint de error desde un navegador, CSP previene que se inyecte contenido.

#### 5. Métricas (Prometheus) — OBLIGATORIO

*   **Qué hace**: Mide automáticamente cada request: contador de requests, histograma de latencia, desglosado por método, path y status code.
*   **Por qué es necesario**: Sin métricas no hay dashboards, no hay alertas, no hay SLOs. Es la base de toda observabilidad cuantitativa.
*   **Debe ser un middleware**: Así cubre todos los endpoints sin tener que instrumentar cada handler manualmente.
*   **Cuidado con la cardinalidad**: No usar el path con IDs dinámicos como label (`/users/12345`). Normalizar a `/users/:id` o las métricas explotarán en cardinalidad.

#### 6. CORS (Cross-Origin Resource Sharing) — CONDICIONAL

*   **Qué hace**: Controla qué orígenes (dominios) pueden hacer requests al API desde un navegador.
*   **Cuándo es necesario**: Cuando el API es consumido por un frontend en un dominio diferente (ej: `app.example.com` consume `api.example.com`).
*   **Cuándo NO es necesario**: APIs consumidas exclusivamente por backends (servicio-a-servicio), mobile apps nativas, o CLIs. El navegador es el único que enforce CORS.
*   **Configuración**:
    *   **Desarrollo**: `AllowAllOrigins: true` está bien para facilitar el trabajo.
    *   **Producción**: Listar explícitamente los orígenes permitidos. **Nunca** `AllowAllOrigins: true` en producción.
    *   Headers permitidos: Incluir los custom headers que tu API usa (`X-API-Key`, `Idempotency-Key`, etc.) además de los estándar (`Content-Type`, `Authorization`).
*   **Preflight requests**: El navegador envía un `OPTIONS` antes de requests con custom headers o métodos no simples. El middleware debe responder correctamente o el request real nunca se ejecutará.

#### 7. Body Size Limit — OBLIGATORIO

*   **Qué hace**: Limita el tamaño máximo del cuerpo de la request (ej: 1 MB).
*   **Por qué es necesario**: Sin este límite, un atacante (o un cliente con un bug) puede enviar un body de 10 GB y agotar la memoria del servidor. Es un vector de ataque de denegación de servicio (DoS) trivial.
*   **Valor recomendado**: Depende del caso de uso. Para APIs JSON típicas, 1 MB es suficiente. Para uploads de archivos, ajustar al máximo esperado y usar streaming.
*   **Comportamiento al exceder**: Retornar `413 Payload Too Large` con un mensaje claro.

#### 8. Content-Type Enforcement — OBLIGATORIO

*   **Qué hace**: Rechaza requests que no envíen el `Content-Type` esperado (generalmente `application/json` para APIs REST).
*   **Por qué es necesario**: Previene que el servidor procese payloads inesperados. Un form HTML que haga POST a tu API enviaría `application/x-www-form-urlencoded` — sin esta validación, podrías intentar parsear basura como JSON y exponer errores internos.
*   **Dónde aplicar**: Solo en métodos que tienen body (`POST`, `PUT`, `PATCH`). No aplicar en `GET` o `DELETE`.
*   **Comportamiento al fallar**: Retornar `415 Unsupported Media Type`.

#### 9. Rate Limiter — OBLIGATORIO

*   **Qué hace**: Limita la cantidad de requests por cliente en un período de tiempo.
*   **Por qué es necesario**: Protege contra abuso, scraping agresivo, ataques de fuerza bruta, y bugs de clientes que hacen retry loops infinitos.
*   **Tipos de rate limiting**:

    | Tipo | Cómo funciona | Cuándo usarlo |
    |------|---------------|---------------|
    | **Token Bucket** | Permite ráfagas (burst) hasta un límite, luego limita al rate constante | Caso general — balancea flexibilidad y protección |
    | **Fixed Window** | Cuenta requests en ventanas fijas (ej: 100 req/minuto) | Simple de implementar, pero permite ráfagas al borde de la ventana |
    | **Sliding Window** | Ventana que se mueve continuamente | Más preciso que fixed window, evita el problema del borde |
    | **Leaky Bucket** | Procesa requests a rate constante, encola el exceso | Cuando necesitas rate de salida estrictamente constante |

*   **Por qué se identifica al cliente**: Normalmente por IP, por API Key, o por usuario (claim del JWT). La elección depende del tipo de autenticación.
*   **Comportamiento al exceder**: Retornar `429 Too Many Requests` con el header `Retry-After` indicando cuántos segundos esperar.
*   **Consideración**: En arquitecturas con múltiples instancias, un rate limiter in-memory solo protege por instancia. Para rate limiting global, usar un store compartido (Redis).

#### 10. Request Timeout — OBLIGATORIO

*   **Qué hace**: Establece un tiempo máximo para procesar cada request. Si se excede, cancela la operación.
*   **Por qué es necesario**: Sin timeout, un handler que espera una respuesta de un servicio externo que nunca llega mantiene la conexión abierta indefinidamente, consumiendo recursos del servidor hasta agotarlos.
*   **Dónde configurar**: A nivel del `http.Server` (Go: `ReadTimeout`, `WriteTimeout`, `IdleTimeout`) y/o como middleware con `context.WithTimeout`.
*   **Valores recomendados**:

    | Timeout | Valor típico | Qué protege |
    |---------|-------------|-------------|
    | `ReadHeaderTimeout` | 5s | Slow headers attack (cliente envía headers lentamente) |
    | `ReadTimeout` | 15s | Lectura completa del request (headers + body) |
    | `WriteTimeout` | 15s | Tiempo para generar y enviar la respuesta |
    | `IdleTimeout` | 60s | Conexiones keep-alive inactivas |

*   **Propagación**: Pasar el `context` del request a todas las operaciones downstream (queries DB, llamadas HTTP externas). Así, cuando el timeout del request se cumple, todo se cancela en cascada.

#### 11. Autenticación — OBLIGATORIO (en rutas protegidas)

*   **Qué hace**: Valida la identidad del llamador (JWT, API Key, OAuth, etc.).
*   **Dónde aplicar**: Solo en rutas que requieren autenticación. Endpoints como `/health`, `/metrics`, y documentación pública NO deben requerir auth.
*   **Detalle**: Ver [Sección 5.4 — Autenticación y Autorización](#54-autenticación-y-autorización).

#### 12. Compresión (gzip) — RECOMENDADO

*   **Qué hace**: Comprime las respuestas antes de enviarlas al cliente, reduciendo el tamaño de transferencia.
*   **Cuándo es necesario**: Cuando las respuestas son grandes (listas de datos, reportes, exports). Para respuestas pequeñas (< 1 KB), la compresión puede ser contraproducente (overhead de CPU sin ganancia real de tamaño).
*   **Cuándo NO usarlo**: Si hay un reverse proxy delante (nginx, Cloudflare, AWS ALB) que ya comprime las respuestas. Comprimir dos veces no reduce más el tamaño y desperdicia CPU.
*   **Configuración**: Solo comprimir si el cliente lo soporta (header `Accept-Encoding: gzip`). Establecer un tamaño mínimo de respuesta para activar la compresión (ej: solo si > 1 KB).

#### 13. Request Tracing (OpenTelemetry) — RECOMENDADO

*   **Qué hace**: Crea un span para cada request HTTP y propaga el trace context a operaciones downstream.
*   **Cuándo es necesario**: Cuando tienes más de un servicio (microservicios). Para un monolito simple puede ser excesivo.
*   **Detalle**: Ver [Sección 9.3 — Tracing Distribuido](#93-tracing-distribuido).

### 16.2 Orden Recomendado del Stack de Middlewares

El orden importa. Este es el orden recomendado para la cadena de middlewares:

```
Request entrante
    │
    ▼
1.  Recovery / Panic Handler     ← Captura cualquier panic de los middlewares y handlers siguientes
    │
    ▼
2.  Request ID                   ← Genera/extrae ID para correlación en todos los logs
    │
    ▼
3.  Logger                       ← Loguea request + response (con request ID ya disponible)
    │
    ▼
4.  Security Headers             ← Agrega headers de protección a toda respuesta
    │
    ▼
5.  Métricas (Prometheus)        ← Mide latencia y status de toda request
    │
    ▼
6.  CORS                         ← Responde preflight OPTIONS antes de cualquier otra validación
    │
    ▼
7.  Body Size Limit              ← Rechaza payloads grandes antes de leer el body
    │
    ▼
8.  Content-Type Enforcement     ← Rechaza content-types no soportados antes de intentar parsear
    │
    ▼
9.  Rate Limiter                 ← Limita requests antes de ejecutar lógica costosa
    │
    ▼
10. Autenticación                ← Valida identidad (JWT / API Key)
    │
    ▼
11. Handler (lógica de negocio)
```

**¿Por qué este orden?**

*   **Recovery primero**: Si cualquier middleware posterior hace panic, recovery lo captura.
*   **Request ID y Logger antes de todo lo demás**: Así cualquier error (rate limit, auth failure, etc.) ya tiene request ID en el log.
*   **Security headers y métricas antes de CORS**: Toda respuesta (incluyendo preflight) lleva headers de seguridad y se mide.
*   **CORS antes de body limit y rate limiter**: Las requests OPTIONS de preflight no tienen body ni deben consumir rate limit.
*   **Rate limiter antes de auth**: Limitar primero evita que un atacante haga fuerza bruta contra el endpoint de autenticación sin restricción.
*   **Auth al final**: Solo se ejecuta si la request ya pasó todas las validaciones previas.

### 16.3 Validación de Input

Toda data que entra al sistema desde el exterior es potencialmente maliciosa o incorrecta. La validación se aplica en diferentes puntos según el tipo de input.

#### Tipos de validación

| Tipo | Dónde se valida | Ejemplo | Qué verificar |
|------|-----------------|---------|----------------|
| **Path params** | Handler / Router | `GET /users/:id` | Formato válido (UUID, integer), no vacío |
| **Query params** | Handler | `?page=1&limit=50` | Tipo correcto, rango válido (limit ≤ 100), valores permitidos para enums |
| **Headers** | Middleware o Handler | `Content-Type`, `Idempotency-Key` | Presencia, formato esperado |
| **Body (JSON)** | Capa de aplicación | `{"email": "user@example.com"}` | Campos requeridos, tipos, formatos, rangos, reglas de negocio |
| **Archivos** | Handler | Upload de imagen | Tamaño máximo, tipo MIME real (no confiar en extension), dimensiones |

#### Dónde validar qué (por capa)

```
HTTP Handler (infraestructura)
├── ¿Headers correctos? (Content-Type, Idempotency-Key)
├── ¿Path params válidos? (UUID bien formado)
├── ¿Query params en rango? (page > 0, limit ≤ 100)
└── ¿JSON parseable? (sintaxis válida)

Capa de Aplicación (use case / validación)
├── ¿Campos requeridos presentes? (email no vacío)
├── ¿Formatos correctos? (email contiene @, teléfono es numérico)
├── ¿Valores en rango? (amount > 0, age entre 0 y 150)
└── ¿Reglas de negocio? (no puedes cancelar un pedido ya entregado)

Capa de Dominio (entidad)
├── ¿Invariantes del dominio? (tipo de evento válido, idempotency key no vacía)
└── Rechaza estados inválidos en construcción (constructor valida)
```

#### Principios de validación

*   **Fail fast**: Validar lo antes posible. No ejecutar queries a la DB si el input ya se sabe inválido.
*   **Mensajes de error útiles**: Indicar qué campo falló y por qué. `"email: must contain @"` es útil. `"invalid request"` no lo es.
*   **Validar en el backend siempre**: Aunque el frontend valide, el backend **debe** validar de nuevo. El frontend es una conveniencia para el usuario; el backend es la fuente de verdad.
*   **No confiar en el tipo MIME declarado**: Para uploads, verificar el contenido real del archivo (magic bytes), no solo el header `Content-Type`.
*   **Sanitizar para el contexto de uso**: No existe un "sanitizar universal". Para SQL se usan queries parametrizadas, para HTML se escapan caracteres, para URLs se codifica. Cada contexto tiene su propia estrategia.

### 16.4 Graceful Shutdown

*   **Qué es**: Cuando el servidor recibe una señal de apagado (SIGTERM, SIGINT), deja de aceptar nuevas conexiones pero espera a que las requests en curso terminen antes de cerrar.
*   **Por qué es obligatorio**: Sin graceful shutdown, las requests en vuelo se cortan abruptamente → respuestas 502 para los clientes → posible corrupción de datos si una escritura a DB se interrumpe a mitad.
*   **Cómo funciona**:
    1.  El proceso recibe `SIGTERM` (señal que Kubernetes envía antes de matar un pod)
    2.  El servidor deja de aceptar nuevas conexiones
    3.  Espera hasta un timeout (ej: 15s) para que las requests activas terminen
    4.  Cierra conexiones a DB, flushea logs, libera recursos
    5.  El proceso termina con exit code 0
*   **Timeout de shutdown**: Debe ser **menor** que el `terminationGracePeriodSeconds` de Kubernetes (default 30s). Si tu shutdown timeout es 15s y K8s da 30s, hay margen. Si es mayor, K8s mata el proceso con SIGKILL.

### 16.5 TLS / HTTPS

*   **En producción**: El tráfico **debe** ir encriptado. No hay excepciones.
*   **Dónde terminar TLS**:

    | Opción | Cómo funciona | Cuándo usarla |
    |--------|---------------|---------------|
    | **En el load balancer / reverse proxy** | Nginx, AWS ALB, o Cloudflare terminan TLS. El backend recibe HTTP plano | La más común. Simplifica el backend, centraliza certificados |
    | **En la aplicación directamente** | El backend sirve HTTPS con cert y key propios | Cuando no hay proxy delante, o por requisitos de compliance (encripción end-to-end) |
    | **mTLS (mutual TLS)** | Ambos lados presentan certificado. El servidor valida al cliente y viceversa | Comunicación servicio-a-servicio en entornos de alta seguridad |

*   **Certificados**: Usar Let's Encrypt (gratuito) o el certificado que provea tu cloud provider. Renovación automática.
*   Si TLS se termina en el proxy, asegurar que la comunicación proxy → backend sea en una red privada (VPC, red interna de Kubernetes).

### 16.6 Checklist de Producción

Resumen rápido de todo lo que un API backend debe tener antes de ir a producción:

| Categoría | Requisito | Obligatorio | Sección |
|-----------|-----------|:-----------:|---------|
| **Middleware** | Recovery / Panic handler | Si | 16.1.1 |
| **Middleware** | Request ID | Si | 16.1.2 |
| **Middleware** | Logger por request | Si | 16.1.3 |
| **Middleware** | Security headers | Si | 16.1.4 |
| **Middleware** | Métricas Prometheus | Si | 16.1.5 |
| **Middleware** | CORS | Solo si hay frontend en otro dominio | 16.1.6 |
| **Middleware** | Body size limit | Si | 16.1.7 |
| **Middleware** | Content-Type enforcement | Si | 16.1.8 |
| **Middleware** | Rate limiting | Si | 16.1.9 |
| **Middleware** | Request timeout | Si | 16.1.10 |
| **Middleware** | Autenticación | Si (en rutas protegidas) | 16.1.11 |
| **Middleware** | Compresión gzip | Si no hay proxy que lo haga | 16.1.12 |
| **Middleware** | Tracing OpenTelemetry | Si hay microservicios | 16.1.13 |
| **Validación** | Input en todas las capas | Si | 16.3 |
| **Infra** | Graceful shutdown | Si | 16.4 |
| **Infra** | TLS / HTTPS | Si | 16.5 |
| **Infra** | Health checks (liveness + readiness) | Si | 13 |
| **Infra** | Configuración validada al arrancar | Si | 4.5 |
| **Observabilidad** | Logs JSON estructurados | Si | 9.1 |
| **Observabilidad** | Métricas RED | Si | 9.2 |
| **Observabilidad** | Alertas con runbooks | Si | 9.4 |
