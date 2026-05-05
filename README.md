## go-core

`go-core` is a reusable infrastructure foundation for Go services.
It is domain-agnostic and focuses on bootstrap/runtime concerns:
config, lifecycle, transport, logging, metrics, tracing, database, and messaging.

Module path:

`github.com/yogayulanda/go-core`

### Release and upgrade discipline

`go-core` is released as stable `v1.0.0` for Go services.
`v1.0.0` is the first compatibility baseline for semver-governed adoption.
Release discipline is intentionally simple:

- CI baseline is the fast repository gate: `make test`, `make vet`, `make lint`
- local release gate is stronger: `make quality-gate`
- staging release validation uses `make smoke-gate`, load gates, and `make failure-drill`
- public contract changes must update README, relevant docs, tests, and `MIGRATION.md` when upgrade behavior changes
- release builds should set `version.Version`, `version.Commit`, and `version.BuildDate` via `-ldflags`

See:

- `docs/PRODUCTION_SIGNOFF.md`
- `docs/CHANGE_CHECKLIST.md`
- `docs/VERSIONING.md`
- `MIGRATION.md`

### Foundation boundary

`go-core` has two allowed contract classes:

- Generic foundation contracts:
  bootstrap/runtime wiring, transport wrappers, config, lifecycle, infra connectors, technical errors, `dbtx`.
- Platform-standard technical contracts:
  intentionally standardized technical contracts shared by a class of services.

Current platform-standard example:

- `logger.TransactionLog`
- `logger.Logger.LogTransaction(...)`
- `observability` metric `app_transaction_total{service,operation,status}`

These transaction observability contracts are for transaction-oriented services.
They are not a license to move business rules into `go-core`.

### What it provides

- App container + graceful shutdown lifecycle (`app`).
- Config loader + validation from environment variables (`config`).
- Structured logger (`logger`) with:
  - JSON in non-dev environment.
  - colored console in `APP_ENV=dev|local|development`.
  - timezone-aware timestamp encoding (default UTC, configurable via `LOG_TIMEZONE`).
  - sensitive-field masking (keep last 2 chars for sensitive values).
  - `ServiceLog` for normal technical service flow.
  - `DBLog` for DB operational and query-related logging.
  - optional `TransactionLog` for transaction-oriented service monitoring.
- Multi-database initialization (`database`) with named DB map and GORM support.
- Optional Redis cache dependency initialization (`cache/redis`) with fail-fast startup ping and aligned `cache_connect` `ServiceLog`.
- Optional Memcached cache dependency initialization (`cache/memcached`) with fail-fast health check, miss-tolerant readiness probe, and aligned `cache_connect` `ServiceLog`.
- gRPC server wrapper + interceptors (`server/grpc`):
  - recovery
  - request-id
  - auth extraction / JWT verification (configurable)
  - transport-aligned `ServiceLog` emission for request flow and panic recovery
  - request metrics + additive service operation metrics
- gRPC-Gateway wrapper (`server/gateway`) exposing:
  - `GET /health`
  - `GET /ready`
  - `GET /version`
  - `GET /metrics`
  - `GET /debug/pprof/*` (optional via `HTTP_PPROF_ENABLED`)
  - HTTP panic recovery
  - HTTP signature validation (optional via `AUTH_SIGNATURE_ENABLED`)
  - OpenTelemetry HTTP span wrapper (`otelhttp`)
  - transport-aligned request-id propagation, HTTP metrics, service metrics, and `ServiceLog`
- Startup helper (`server`):
  - `Run(...)` to orchestrate gRPC + gateway + lifecycle with centralized error handling.
  - `DescribeFromProto(...)` to list HTTP/gRPC routes from proto descriptors.
  - `LogStartupReadiness(...)` to emit readiness `ServiceLog` for gRPC, gateway, and combined service readiness.
- OTEL tracing bootstrap (`observability`).
- Prometheus metrics (`observability`):
  - `app_http_request_total{service,method,route,status}`
  - `app_http_request_duration_seconds{service,method,route}`
  - `app_request_total{service,method,status}`
  - `app_request_duration_seconds{service,method}`
  - `app_service_operation_total{service,operation,status}`
  - `app_service_operation_duration_seconds{service,operation}`
  - `app_db_operation_total{service,db_name,operation,status}`
  - `app_db_operation_duration_seconds{service,db_name,operation}`
  - `app_message_publish_total{service,topic,status}`
  - `app_message_consume_total{service,topic,group,status}`
  - `app_message_process_duration_seconds{service,topic,group}`
  - `app_outbox_batch_total{service,status}`
  - `app_outbox_batch_duration_seconds{service}`
  - `app_outbox_batch_size{service}`
  - `app_transaction_total{service,operation,status}`
- Kafka publisher/consumer abstraction (`messaging`) with additive logger/metrics options and app-level defaults.
- Outbox helpers (`messaging/outbox`) with driver-aware SQL (`mysql|postgres|sqlserver`), `RunOnce(...)`, and explicit `StartChecked(...)`.
- Goose migration helper (`migration`) including auto-run support.
- DB transaction helper (`dbtx`) with context propagation (`WithTx`, `WithTxOptions`).
- Outbound resilience helper (`resilience`) for timeout + retry policy, circuit breaker (`sony/gobreaker`), plus additive retry/timeout hooks for logger-backed observability.
- Resilient HTTP client (`httpclient`) with built-in:
  - circuit breaker
  - retries with backoff
  - OpenTelemetry tracing
  - structured logging (`ServiceLog`)
- Common app error contract + mapper (`errors`) with stable code and optional validation details.

### Security scope

- By default (`INTERNAL_JWT_ENABLED=false`), go-core extracts generic auth metadata
  from incoming gRPC metadata:
  - `x-subject`
  - `x-session-id`
  - `x-role`
  - `x-claim-<name>` (mapped into `security.Claims.Attributes`)
- If `INTERNAL_JWT_ENABLED=true`, go-core enforces bearer JWT verification in gRPC interceptor:
  - RSA signature validation (`RS256/RS384/RS512`) via static key OR dynamic background polled JWKS endpoints
  - standard time claims validation (`exp`, `nbf`, `iat`)
  - optional issuer check (`INTERNAL_JWT_ISSUER`)
  - optional audience check (`INTERNAL_JWT_AUDIENCE`)
- JWT-to-claims mapping:
  - `sub` -> `Claims.Subject`
  - `session_id`/`sid` -> `Claims.SessionID`
  - `role` -> `Claims.Role`
  - `attributes` object -> `Claims.Attributes`
- `INTERNAL_JWT_PUBLIC_KEY` is required when JWT is enabled.
- Optional transport TLS is supported for gRPC and HTTP gateway (`GRPC_TLS_*`, `HTTP_TLS_*`).

Operational notes:

- gRPC startup emits `auth_config` so operators can confirm whether the service is running in metadata extraction mode or JWT verification mode.
- JWT auth failures are sanitized to clients as unauthorized responses.
- internal service logs keep stable auth failure reasons such as missing authorization header, invalid token, invalid issuer, and invalid audience.

### Configuration

All values are loaded from environment variables.

Configuration profiles:

- see `docs/CONFIGURATION_PROFILES.md` for grouped onboarding guidance
- use `cfg.Validate()` for the compact public error
- use `cfg.ValidateIssues()` for structured validation issues by section and field

Core:

- `SERVICE_NAME` (required)
- `APP_ENV` (default: `dev`)
- `LOG_LEVEL` (default: `info`)
- `LOG_TIMEZONE` (optional IANA TZ name, default: `UTC`; example: `Asia/Jakarta`)
- `SHUTDOWN_TIMEOUT` (default: `10s`)
- `GRPC_PORT` (default: `50051`)
- `HTTP_PORT` (default: `8080`)
- `GRPC_TLS_ENABLED` (default: `false`)
- `GRPC_TLS_CERT_FILE` (required when `GRPC_TLS_ENABLED=true`)
- `GRPC_TLS_KEY_FILE` (required when `GRPC_TLS_ENABLED=true`)
- `HTTP_TLS_ENABLED` (default: `false`)
- `HTTP_TLS_CERT_FILE` (required when `HTTP_TLS_ENABLED=true`)
- `HTTP_TLS_KEY_FILE` (required when `HTTP_TLS_ENABLED=true`)

Databases:

- `DB_LIST` (optional, comma-separated aliases, example: `primary,ledger_history`)
- Per DB name (`<N>` is uppercase name from `DB_LIST`):
  - `DB_<N>_DRIVER` (required; `mysql|postgres|sqlserver`)
  - `DB_<N>_DSN` (optional override)
  - or composed fields:
    - `DB_<N>_HOST`
    - `DB_<N>_PORT`
    - `DB_<N>_NAME`
    - `DB_<N>_USER`
    - `DB_<N>_PASSWORD`
    - `DB_<N>_PARAMS` (optional query params)
  - pool settings (optional):
    - `DB_<N>_REQUIRED` (default: `true`, fail-fast on startup and affects `/ready`)
    - `DB_<N>_MAX_OPEN_CONNS` (default: `20`)
    - `DB_<N>_MAX_IDLE_CONNS` (default: `10`)
    - `DB_<N>_CONN_MAX_IDLE_TIME` (default: `2m`)
    - `DB_<N>_CONN_MAX_LIFETIME` (default: `5m`)

Alias notes:

- aliases come from the consuming service, not from `go-core`
- aliases may contain underscores, for example `transaction_history`
- env lookup uses uppercase alias token, for example `DB_TRANSACTION_HISTORY_DRIVER`
- runtime map keys are normalized to lowercase for deterministic lookup

Transaction naming note:

- `dbtx` refers to SQL transaction orchestration.
- `TransactionLog` refers to transaction-flow monitoring for transaction-oriented services.
- They solve different concerns and are intentionally separate.

Migration:

- `MIGRATION_AUTO_RUN` (default: `false`)
- `MIGRATION_DB` (no default; must exist in `DB_LIST` when auto-run enabled)
- `MIGRATION_DIR` (no default; required when auto-run enabled)
- `MIGRATION_LOCK_ENABLED` (default: `true`)
- `MIGRATION_LOCK_KEY` (default: empty; auto-generated as `<SERVICE_NAME>:migration:<MIGRATION_DB>`)
- `MIGRATION_LOCK_TIMEOUT` (default: `30s`)

When lock is enabled, auto-migration uses DB-native locks to avoid concurrent `goose up` on multi-pod startup (`sp_getapplock` for SQL Server, `GET_LOCK` for MySQL, advisory lock for Postgres).

Migration runtime notes:

- `migration.AutoRunUp(cfg)` remains the compact explicit entry point.
- `migration.AutoRunUpWithLogger(cfg, log)` is available when the service wants startup migration runtime signals through `ServiceLog`.
- logger-aware autorun emits `migration_autorun` and `migration_lock` without adding hidden startup behavior.

Observability:

- `OTEL_EXPORTER_OTLP_ENDPOINT` (optional)
- `OTEL_EXPORTER_OTLP_INSECURE` (default: `false`; set `true` only for local/non-TLS collector)
- `OTEL_EXPORTER_OTLP_CA_CERT_FILE` (optional custom CA for OTLP TLS)
- `TRACE_SAMPLING_RATIO` (default: `0.1`)

Redis:

- `REDIS_ENABLED` (default: `false`)
- `REDIS_ADDRESS` (required if enabled)
- `REDIS_PASSWORD`
- `REDIS_DB` (default: `0`)

Behavior:

- enabling Redis means the service has chosen Redis as a required runtime dependency
- Redis initialization is fail-fast during `app.New(...)`
- `/ready` reports Redis as required when enabled and returns `503` if Redis health fails

Memcached:

- `MEMCACHED_ENABLED` (default: `false`)
- `MEMCACHED_SERVERS` (comma-separated, required if enabled)
- `MEMCACHED_ADDRESS` (single address fallback, optional)
- `MEMCACHE_HOST` (legacy host fallback, default: empty)
- `MEMCACHE_PORT` (legacy port fallback, default: `11211`)
- `MEMCACHED_TIMEOUT` (default: `2s`)

Behavior:

- enabling Memcached means the service has chosen Memcached as a required runtime dependency
- Memcached initialization is fail-fast during `app.New(...)`
- Memcached health uses a bounded `Get(...)` probe where `cache miss` is treated as healthy by design
- `/ready` reports Memcached as required when enabled and returns `503` if Memcached health fails

Kafka:

- `KAFKA_ENABLED` (default: `false`)
- `KAFKA_BROKERS` (required if enabled; comma-separated)
- `KAFKA_CLIENT_ID`
- `KAFKA_USERNAME` (SASL Plain username)
- `KAFKA_PASSWORD` (SASL Plain password)
- `KAFKA_JKS_FILE` (Path to JKS certificate file)
- `KAFKA_JKS_PASSWORD` (Password for JKS file)

Auth:

- `INTERNAL_JWT_ENABLED`
- `INTERNAL_JWT_PUBLIC_KEY` (used as static key if JWKS is not configured)
- `INTERNAL_JWT_JWKS_ENDPOINT` (enables dynamic background JWKS fetching via keyfunc)
- `INTERNAL_JWT_JWKS_REFRESH_INTERVAL` (default: `1h`)
- `INTERNAL_JWT_ISSUER`
- `INTERNAL_JWT_AUDIENCE`
- `INTERNAL_JWT_LEEWAY` (default: `30s`)
- `INTERNAL_JWT_INCLUDE_METHODS` (optional, comma-separated gRPC full methods)
- `INTERNAL_JWT_EXCLUDE_METHODS` (optional, comma-separated gRPC full methods)

- `AUTH_SIGNATURE_ENABLED` (default: `false`, enables HTTP payload signature verification)
- `AUTH_SIGNATURE_MASTER_KEY` (secret key used for HMAC-SHA256 signature verification)
- `AUTH_SIGNATURE_HEADER_KEY` (default: `x-signature`)
- `AUTH_SIGNATURE_TIMESTAMP_KEY` (default: `x-timestamp`)
- `AUTH_SIGNATURE_MAX_TIME_DRIFT` (default: `5m`, prevents replay attacks)

Method policy notes:

- If `INTERNAL_JWT_INCLUDE_METHODS` is set, only listed methods enforce JWT.
- If include list is empty, all methods enforce JWT except those in exclude list.
- `INTERNAL_JWT_INCLUDE_METHODS` and `INTERNAL_JWT_EXCLUDE_METHODS` cannot be used together.

### Recommended baseline env (production-like)

```env
SERVICE_NAME=transaction-history-service
APP_ENV=production
LOG_LEVEL=info
SHUTDOWN_TIMEOUT=10s
GRPC_PORT=9090
HTTP_PORT=8080

# Database (example)
DB_LIST=primary
DB_PRIMARY_DRIVER=sqlserver
DB_PRIMARY_HOST=127.0.0.1
DB_PRIMARY_PORT=1433
DB_PRIMARY_NAME=app_db
DB_PRIMARY_USER=sa
DB_PRIMARY_PASSWORD=********
DB_PRIMARY_REQUIRED=true
DB_PRIMARY_CONN_MAX_IDLE_TIME=2m

MIGRATION_AUTO_RUN=true
MIGRATION_DB=primary
MIGRATION_DIR=migrations/primary

# Internal JWT
INTERNAL_JWT_ENABLED=true
INTERNAL_JWT_PUBLIC_KEY=/etc/secrets/internal-jwt-public.pem
INTERNAL_JWT_ISSUER=internal-auth
INTERNAL_JWT_AUDIENCE=internal-services
INTERNAL_JWT_LEEWAY=30s
# Choose one:
# INTERNAL_JWT_INCLUDE_METHODS=/history.v1.HistoryService/CreateTransactionHistory
# INTERNAL_JWT_EXCLUDE_METHODS=/grpc.health.v1.Health/Check,/grpc.health.v1.Health/Watch

# Optional dependencies
REDIS_ENABLED=false
MEMCACHED_ENABLED=false
KAFKA_ENABLED=false
```

### Error response contract

HTTP error response (gateway) is kept strictly structured and standardized:

```json
{
  "success": false,
  "code": "TRF-VAL-001",
  "message": "invalid request",
  "user_message": "User friendly message",
  "trace_id": "req-123",
  "transaction_id": "tx-123",
  "timestamp": "2026-05-05T17:00:00Z",
  "details": [
    {"field": "user_id", "reason": "required"}
  ]
}
```

Notes:

- `details` is optional, typically used for validation errors.
- The `code` uses a strict `<DOMAIN>-<CATEGORY>-<NUMBER>` formatting logic.
- gRPC mapper automatically packs extended attributes (`domain`, `user_message`, `retryable`, `finality`) into the `ErrorInfo.Metadata`.
- HTTP Gateway extracts `trace_id` automatically from OTEL trace span, and `transaction_id` from observability context.
- Unknown external `ErrorInfo.reason` values are sanitized and fallback to gRPC status mapping.

#### Building Application Errors

Downstream services should build errors using the `ErrorBuilder` to ensure correct taxonomy and categorization:

```go
import coreErrors "github.com/yogayulanda/go-core/errors"

var ErrInvalidAccount = coreErrors.Build("TRF", coreErrors.CategoryVAL, "001").
	Message("dest_account_number length is strictly 10 digits"). // Technical log
	UserMessage("Nomor rekening tujuan tidak valid.").             // Safe for frontend
	Finality(coreErrors.FinalityBusiness).                       // E.g., Business, TechnicalRecoverable
	Done()
```

Categories: `VAL` (Validation), `AUTH` (Auth), `SES` (Session), `SWI` (Switch/Partner), `DB` (Database), `REC` (Recoverable/Technical).

### Readiness behavior

`GET /ready` returns JSON with per-component checks.
HTTP status is:

- `200` when all required dependencies are ready.
- `503` when any required dependency is not ready.

Required dependencies:

- all required databases (`DB_<NAME>_REQUIRED=true`)
- Redis (if `REDIS_ENABLED=true`)
- Memcached (if `MEMCACHED_ENABLED=true`)
- Kafka broker reachability (if `KAFKA_ENABLED=true`)

Cache notes:

- enabling Redis or Memcached is an explicit service choice, not passive configuration
- cache initialization is fail-fast during app bootstrap
- cache runtime startup emits `ServiceLog` with `operation=cache_connect`

Example response:

```json
{
  "status": "not_ready",
  "checks": {
    "database.primary": {"status": "up", "required": true},
    "redis": {"status": "down", "required": true, "message": "health check failed"},
    "memcached": {"status": "skipped", "required": false, "message": "disabled"},
    "kafka": {"status": "skipped", "required": false, "message": "disabled"}
  }
}
```

If your service does not use database, you can keep `DB_LIST` empty.
When `MIGRATION_AUTO_RUN=true`, `MIGRATION_DB` and `MIGRATION_DIR` must be set explicitly, and `MIGRATION_DB` must exist in `DB_LIST`.

### Golden path for a new service

Canonical startup flow:

1. `config.Load(...)`
2. `cfg.Validate()`
3. optional `migration.AutoRunUp(cfg)`
4. `app.New(ctx, cfg)`
5. build gRPC and/or gateway transport
6. `server.Run(ctx, application, ...)`

Use:

- `errors.AppError` for service error contract
- `dbtx.WithTx(...)` for SQL transaction orchestration
- `logger.ServiceLog` for structured service-flow logging
- `logger.DBLog` for structured DB logging when the service touches a database
- `TransactionLog` only when the service belongs to the transaction-oriented class
- Redis, Memcached, Kafka, and migration only when the service explicitly chooses them
- rely on `server.Run(...)` lifecycle/service logs for startup, shutdown, and component failure orchestration
- rely on gateway/gRPC transport wrappers for aligned request ID, request metrics, and additive service metrics
- rely on `app.NewKafkaPublisher(...)` / `app.NewKafkaConsumer(...)` for default messaging logger + metrics wiring when Kafka is enabled
- keep outbox worker startup explicit through `outbox.Worker.StartChecked(ctx)` or service-controlled `RunOnce(ctx)`

### Logging flavors

`go-core` supports three intentional logging flavors:

- `ServiceLog`:
  standard structured log for normal technical service flow
- `DBLog`:
  standard structured log for DB connect/ping/query/timeout/failure reporting
- `TransactionLog`:
  platform-standard structured log for transaction-oriented services

Keep using `Info/Error/Debug/Warn` for flexible low-level technical logs and framework internals.
Keep using `EventLog` for important event/compliance-style logging.

Runtime and transport alignment now means:

- `app.New(...)`, `app.Start(...)`, lifecycle shutdown, and `server.Run(...)` emit structured `ServiceLog` for orchestration milestones.
- `server.LogStartupReadiness(...)` emits readiness `ServiceLog` instead of ad hoc startup strings.
- gRPC request flow emits both request metrics and additive service metrics under `grpc_request`.
- HTTP gateway flow emits both HTTP metrics and additive service metrics under `http_request`.
- publisher flow emits `message_publish` service logs plus `app_message_publish_total`.
- consumer flow emits `message_consume` service logs plus consume/process metrics.
- outbox worker emits `outbox_worker` and `outbox_batch` service logs plus outbox batch metrics.

### Minimal integration flow in a service

```go
package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	coreapp "github.com/yogayulanda/go-core/app"
	coreconfig "github.com/yogayulanda/go-core/config"
	coremigration "github.com/yogayulanda/go-core/migration"
	coreserver "github.com/yogayulanda/go-core/server"
	coregateway "github.com/yogayulanda/go-core/server/gateway"
	coregrpc "github.com/yogayulanda/go-core/server/grpc"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg, err := coreconfig.Load(coreconfig.WithDotEnv(".env"))
	if err != nil {
		log.Fatal(err)
	}
	if err := cfg.Validate(); err != nil {
		log.Fatal(err)
	}
	if err := coremigration.AutoRunUp(cfg); err != nil {
		log.Fatal(err)
	}

	application, err := coreapp.New(ctx, cfg)
	if err != nil {
		log.Fatal(err)
	}
	grpcServer, err := coregrpc.New(application)
	if err != nil {
		log.Fatal(err)
	}
	gatewayServer, err := coregateway.New(application, func(ctx context.Context, mux *runtime.ServeMux) error {
		// register grpc-gateway handlers here
		return nil
	})
	if err != nil {
		log.Fatal(err)
	}

	grpcServer.Register(func(s *grpc.Server) {
		// register grpc service handlers here
	})

	go coreserver.LogStartupReadiness(ctx, application.Logger(), cfg.GRPC.Port, cfg.HTTP.Port, 10*time.Second, cfg.HTTP.TLSEnabled)

	if err := coreserver.Run(ctx, application, grpcServer, gatewayServer); err != nil {
		log.Fatal(err)
	}
}
```

More guidance:

- `docs/SERVICE_BOOTSTRAP.md`
- `docs/TRANSACTION_OBSERVABILITY.md`
- `docs/FOUNDATION_BOUNDARY.md`
- `docs/MESSAGING_PATTERN.md`
- `docs/CONFIGURATION_PROFILES.md`

### Quality checks

Install linter:

```bash
go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.64.8
```

Run checks:

```bash
make test
make vet
make lint
# or run all:
make check
```

CI:

- `.github/workflows/ci.yml` runs `go test ./...`, `go vet ./...`, and `golangci-lint run`

Foundation repo change discipline:

- review `docs/CHANGE_CHECKLIST.md`
- update `MIGRATION.md` whenever public upgrade behavior changes
- update `CHANGELOG.md` for each tagged release

### Production sign-off

Detailed checklist:
- `docs/PRODUCTION_SIGNOFF.md`

Evidence template:
- `docs/RELEASE_EVIDENCE_TEMPLATE.md`

Gate commands:

```bash
# full local quality/security gate
make quality-gate

# staging smoke gate
BASE_URL=https://staging.example.com make smoke-gate

# staging load gates (requires k6)
BASE_URL=https://staging.example.com TARGET_PATH=/v1/your-endpoint make load-steady
BASE_URL=https://staging.example.com TARGET_PATH=/v1/your-endpoint make load-spike
BASE_URL=https://staging.example.com TARGET_PATH=/v1/your-endpoint make load-soak

# staging failure drill (example with kubectl)
BASE_URL=https://staging.example.com \
STOP_DB_CMD="kubectl scale deploy/db --replicas=0 -n staging" \
START_DB_CMD="kubectl scale deploy/db --replicas=1 -n staging" \
STOP_KAFKA_CMD="kubectl scale sts/kafka --replicas=0 -n staging" \
START_KAFKA_CMD="kubectl scale sts/kafka --replicas=1 -n staging" \
make failure-drill
```

### Version metadata

Set build-time values with `-ldflags` for `/version` endpoint data:

```bash
go build -ldflags "\
  -X 'github.com/yogayulanda/go-core/version.Version=1.0.0' \
  -X 'github.com/yogayulanda/go-core/version.Commit=$(git rev-parse HEAD)' \
  -X 'github.com/yogayulanda/go-core/version.BuildDate=$(date -u +%Y-%m-%dT%H:%M:%SZ)'"
```

### Release notes and changelog

- use GitHub Release for announcement-style release notes
- use `CHANGELOG.md` for repository version history
- use `docs/RELEASING.md` for the repeatable release process
