## go-core

`go-core` is a reusable infrastructure foundation for Go services.
It is domain-agnostic and focuses on bootstrap/runtime concerns:
config, lifecycle, transport, logging, metrics, tracing, database, and messaging.

Module path:

`github.com/yogayulanda/go-core`

### What it provides

- App container + graceful shutdown lifecycle (`app`).
- Config loader + validation from environment variables (`config`).
- Structured logger (`logger`) with:
  - JSON in non-dev environment.
  - colored console in `APP_ENV=dev|local|development`.
  - timezone-aware timestamp encoding (default UTC, configurable via `LOG_TIMEZONE`).
  - sensitive-field masking (keep last 2 chars for sensitive values).
- Multi-database initialization (`database`) with named DB map.
- Optional Redis cache initialization (`cache/redis`).
- Optional Memcached cache initialization (`cache/memcached`).
- gRPC server wrapper + interceptors (`server/grpc`):
  - recovery
  - request-id
  - auth extraction / JWT verification (configurable)
  - logging
  - metrics
- gRPC-Gateway wrapper (`server/gateway`) exposing:
  - `GET /health`
  - `GET /ready`
  - `GET /version`
  - `GET /metrics`
- Startup helper (`server`):
  - `Run(...)` to orchestrate gRPC + gateway + lifecycle with centralized error handling.
  - `DescribeFromProto(...)` to list HTTP/gRPC routes from proto descriptors.
  - `LogStartupReadiness(...)` to log gRPC and gateway readiness.
- OTEL tracing bootstrap (`observability`).
- Prometheus metrics (`observability`):
  - `app_http_request_total{service,method,route,status}`
  - `app_http_request_duration_seconds{service,method,route}`
  - `app_request_total{service,method,status}`
  - `app_request_duration_seconds{service,method}`
  - `app_transaction_total{service,operation,status}`
- Kafka publisher/consumer abstraction (`messaging`).
- Outbox helpers (`messaging/outbox`) with driver-aware SQL (`mysql|postgres|sqlserver`).
- Goose migration helper (`migration`) including auto-run support.
- DB transaction helper (`dbtx`) with context propagation (`WithTx`, `WithTxOptions`).
- Outbound resilience helper (`resilience`) for timeout + retry policy.
- Common app error contract + mapper (`errors`) with stable code and optional validation details.

### Security scope

- By default (`INTERNAL_JWT_ENABLED=false`), go-core extracts generic auth metadata
  from incoming gRPC metadata:
  - `x-subject`
  - `x-session-id`
  - `x-role`
  - `x-claim-<name>` (mapped into `security.Claims.Attributes`)
- If `INTERNAL_JWT_ENABLED=true`, go-core enforces bearer JWT verification in gRPC interceptor:
  - RSA signature validation (`RS256/RS384/RS512`)
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

### Configuration

All values are loaded from environment variables.

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

- `DB_LIST` (optional, comma-separated names, example: `transaction,history`)
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
    - `DB_<N>_CONN_MAX_LIFETIME` (default: `5m`)

Migration:

- `MIGRATION_AUTO_RUN` (default: `false`)
- `MIGRATION_DB` (default: `transaction`, must exist in `DB_LIST` when auto-run enabled)
- `MIGRATION_DIR` (default: `migrations/transaction`)
- `MIGRATION_LOCK_ENABLED` (default: `true`)
- `MIGRATION_LOCK_KEY` (default: empty; auto-generated as `<SERVICE_NAME>:migration:<MIGRATION_DB>`)
- `MIGRATION_LOCK_TIMEOUT` (default: `30s`)

When lock is enabled, auto-migration uses DB-native locks to avoid concurrent `goose up` on multi-pod startup (`sp_getapplock` for SQL Server, `GET_LOCK` for MySQL, advisory lock for Postgres).

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

Memcached:

- `MEMCACHED_ENABLED` (default: `false`)
- `MEMCACHED_SERVERS` (comma-separated, required if enabled)
- `MEMCACHED_ADDRESS` (single address fallback, optional alternative to `MEMCACHED_SERVERS`)
- `MEMCACHED_TIMEOUT` (default: `2s`)

Kafka:

- `KAFKA_ENABLED` (default: `false`)
- `KAFKA_BROKERS` (required if enabled; comma-separated)
- `KAFKA_CLIENT_ID`

Auth:

- `INTERNAL_JWT_ENABLED`
- `INTERNAL_JWT_PUBLIC_KEY`
- `INTERNAL_JWT_ISSUER`
- `INTERNAL_JWT_AUDIENCE`
- `INTERNAL_JWT_LEEWAY` (default: `30s`)
- `INTERNAL_JWT_INCLUDE_METHODS` (optional, comma-separated gRPC full methods)
- `INTERNAL_JWT_EXCLUDE_METHODS` (optional, comma-separated gRPC full methods)

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
DB_LIST=transaction
DB_TRANSACTION_DRIVER=sqlserver
DB_TRANSACTION_HOST=127.0.0.1
DB_TRANSACTION_PORT=1433
DB_TRANSACTION_NAME=transaction_history
DB_TRANSACTION_USER=sa
DB_TRANSACTION_PASSWORD=********
DB_TRANSACTION_REQUIRED=true

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

HTTP error response (gateway) is kept compact:

```json
{
  "code": "INVALID_REQUEST",
  "message": "invalid request",
  "request_id": "req-123",
  "details": [
    {"field": "user_id", "reason": "required"}
  ]
}
```

Notes:

- `details` is optional, typically used for validation errors.
- Internal classification such as error category is kept in logs, not exposed in API response.
- gRPC mapper keeps stable error code via `ErrorInfo.reason`.
- Unknown external `ErrorInfo.reason` values are sanitized and fallback to gRPC status mapping.

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

Example response:

```json
{
  "status": "not_ready",
  "checks": {
    "database.transaction": {"status": "up", "required": true},
    "redis": {"status": "down", "required": true, "message": "health check failed"},
    "memcached": {"status": "skipped", "required": false, "message": "disabled"},
    "kafka": {"status": "skipped", "required": false, "message": "disabled"}
  }
}
```

If your service does not use database, you can keep `DB_LIST` empty.
When `MIGRATION_AUTO_RUN=true`, `MIGRATION_DB` must exist in `DB_LIST`.

### Minimal integration flow in a service

```go
package main

import (
	"context"
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

	cfg, _ := coreconfig.Load(coreconfig.WithDotEnv(".env"))
	_ = cfg.Validate()
	_ = coremigration.AutoRunUp(cfg)

	core, _ := coreapp.New(ctx, cfg)
	grpcServer, _ := coregrpc.New(core)
	gatewayServer, _ := coregateway.New(core, func(ctx context.Context, mux *runtime.ServeMux) error {
		// register grpc-gateway handlers here
		return nil
	})

	grpcServer.Register(func(s *grpc.Server) {
		// register grpc service handlers here
	})

	go coreserver.LogStartupReadiness(ctx, core.Logger(), cfg.GRPC.Port, cfg.HTTP.Port, 10*time.Second, cfg.HTTP.TLSEnabled)

	_ = coreserver.Run(ctx, core, grpcServer, gatewayServer)
}
```

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
