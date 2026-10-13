# go-core

`go-core` is a reusable Go backend foundation for production-minded services. It provides common runtime building blocks for configuration, lifecycle management, transport wrappers, logging, metrics, tracing, database access, messaging, migrations, resilience, and error handling.

The project is intentionally infrastructure-focused. It is not a service template, product domain, or toy demo. Consuming services own their business logic, persistence models, API contracts, deployment topology, and operational policy.

Module path:

```text
github.com/yogayulanda/go-core
```

## What It Solves

- Consistent service bootstrap and graceful shutdown.
- Environment-driven configuration with validation.
- gRPC and HTTP/gRPC-Gateway server wiring.
- Request IDs, recovery, auth metadata/JWT verification, and middleware/interceptor behavior.
- Structured logging with sensitive-field redaction.
- Prometheus metrics and OpenTelemetry tracing hooks.
- SQL database initialization, migrations, and transaction helpers.
- Kafka publishing/consuming and outbox helpers.
- Redis and Memcached cache initialization.
- Resilience helpers for timeout, opt-in retry, circuit breaker, and HTTP/gRPC clients.
- Stable application error contracts for transport-safe responses.

## Architecture Overview

`go-core` keeps infrastructure concerns separated by package:

- `app/` owns application container setup, dependency initialization, and lifecycle hooks.
- `config/` loads and validates environment-based runtime configuration.
- `server/grpc/` provides gRPC server construction, interceptors, recovery, auth, and metrics.
- `server/gateway/` provides HTTP/gRPC-Gateway setup, envelopes, health/readiness/version/metrics endpoints, pprof, CORS, signature validation, and panic recovery.
- `database/` opens configured SQL databases through GORM.
- `dbtx/` provides explicit SQL transaction propagation helpers.
- `migration/` wraps Goose migration execution and optional startup auto-run.
- `messaging/` provides Kafka publisher/consumer abstractions.
- `messaging/outbox/` provides driver-aware SQL outbox helpers.
- `logger/` defines structured technical logging contracts and redaction.
- `observability/` contains tracing, metrics, request identity, and runtime diagnostic snapshots.
- `errors/` defines application error taxonomy and gRPC/HTTP mapping.
- `cache/`, `httpclient/`, `resilience/`, `security/`, and `version/` cover supporting runtime concerns.

The intended service shape remains conventional Go:

1. Transport handlers translate requests into service calls.
2. Service/domain code owns business rules.
3. Repository code owns persistence access.
4. `go-core` supplies shared runtime, transport, observability, and infrastructure glue.

## Tech Stack

- Go 1.24+
- gRPC and grpc-gateway
- GORM with SQL Server and MySQL support
- Goose migrations
- Kafka via `segmentio/kafka-go`
- Redis and Memcached clients
- Prometheus metrics
- OpenTelemetry tracing
- Zap logging
- JWT verification with static RSA public key or JWKS
- Resty-based resilient HTTP client

## Development Setup

Requirements:

- Go 1.24 or newer
- `make`
- Optional: `golangci-lint` for linting
- Optional: `k6` for load-gate scripts

Clone and verify:

```bash
git clone https://github.com/yogayulanda/go-core.git
cd go-core
go mod download
make test
make vet
```

Install the linter used by the repository:

```bash
go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.64.8
make lint
```

Run the standard local gate:

```bash
make check
```

Run the stronger release-oriented local gate:

```bash
make quality-gate
```

## Environment Configuration

Configuration is loaded from environment variables. For local development, copy `.env.example` to `.env` and adjust values for your local dependencies.

Core variables:

- `SERVICE_NAME`
- `APP_ENV`
- `LOG_LEVEL`
- `GRPC_PORT`
- `HTTP_PORT`
- `HTTP_CORS_ALLOWED_ORIGINS`
- `HTTP_PPROF_ENABLED`
- `DIAG_ENABLED`
- `METRICS_ENABLED` (default false; enables metric recording and `/metrics`)

Database variables:

- `DB_LIST`
- `DB_<NAME>_DRIVER` supports `sqlserver` and `mysql`
- `DB_<NAME>_HOST`
- `DB_<NAME>_PORT`
- `DB_<NAME>_NAME`
- `DB_<NAME>_USER`
- `DB_<NAME>_PASSWORD`
- `DB_<NAME>_PARAMS`

Optional runtime dependencies:

- Redis: `REDIS_ADDRESS` enables Redis; `REDIS_PASSWORD` is optional
- Memcached: `MEMCACHED_SERVERS` enables Memcached
- Kafka: `KAFKA_BROKERS` enables Kafka; optional SASL uses `KAFKA_USERNAME`, `KAFKA_PASSWORD`, and `KAFKA_SASL_MECHANISM`
- Kafka TLS: `KAFKA_JKS_FILE` and `KAFKA_JKS_PASSWORD`
- Tracing: `APM_ENDPOINT` enables the existing OTLP gRPC exporter
- HTTP signatures: a non-empty `AUTH_SIGNATURE_MASTER_KEY` enables signature validation

See `docs/CONFIGURATION_PROFILES.md` for grouped configuration guidance.

## Minimal Service Bootstrap

```go
package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

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
	grpcServer.Register(func(s *grpc.Server) {
		// Register service implementations here.
	})

	gatewayServer, err := coregateway.New(application, func(ctx context.Context, mux *runtime.ServeMux) error {
		// Register grpc-gateway handlers here.
		return nil
	})
	if err != nil {
		log.Fatal(err)
	}

	if err := coreserver.Run(ctx, application, grpcServer, gatewayServer); err != nil {
		log.Fatal(err)
	}
}
```

## Error Handling Philosophy

External responses should be predictable and safe. Internal details belong in logs and traces, not client payloads.

- Use `errors.New`, `errors.Wrap`, or `errors.Validation` for application-visible failures; register custom response definitions with `errors.Register`.
- Keep validation details explicit and structured.
- Sanitize unknown/internal transport errors to stable public messages.
- Preserve richer technical context through structured logs and observability signals.

## Observability

The repository provides additive observability primitives rather than a mandatory platform:

- structured service, DB, event, and transaction-oriented logs
- sensitive key redaction for common secret fields
- Prometheus metrics for HTTP, gRPC, service operations, DB operations, messaging, outbox, and transaction-oriented flows
- OpenTelemetry tracing bootstrap and transport wrappers
- OpenTelemetry OTLP export to Elastic APM Server via `APM_ENDPOINT`; no Elastic APM agent or `transaction.id` log field
- health, readiness, metrics, version, and optional pprof endpoints
- bounded gateway `route` labels from matched templates; unmatched paths share `unmatched`
- optional diagnostic snapshots when goroutine or cgroup heap thresholds are crossed; `DIAG_ENABLED=false` disables the watcher

Shutdown starts a drain phase: `/ready` returns 503 and gRPC health reports `NOT_SERVING`, then the configured drain delay elapses before HTTP/gRPC graceful shutdown. Defaults are 3 seconds drain and 25 seconds shutdown.

## Outbound Clients

`httpclient.NewClient` does not retry by default. Use `httpclient.WithRetry(...)` to opt in. A received HTTP response, including 5xx, is returned with a nil transport error; connection/timeout failures use `httpclient.TransportError`. Request identity headers are forwarded only with `httpclient.WithInternalPropagation()`. HTTP body logging is disabled unless a masking callback is provided.

`grpcclient.NewClient` provides request identity propagation, OpenTelemetry stats, call logging/metrics, keepalive, and a 30-second default deadline when the supplied context has no deadline. Service-config retries are disabled. The default transport is plaintext to match the default go-core gRPC server; configure credentials explicitly for TLS.

## Kafka And Audit Events

Kafka supports SASL `PLAIN`, `SCRAM-SHA-256`, and `SCRAM-SHA-512`; `KAFKA_JKS_FILE` accepts JKS or PEM trust material. Producer, consumer, and readiness use the same security config. The outbox uses an identity row ID plus UUID `event_id`, per-key leases, bounded retries, and `DEAD` state. `outbox.Publisher.PublishAudit` writes CloudEvents 1.0 `ce_*` attributes (including `ce_sequence`) into the outbox; consumer handlers can read them with `messaging.CloudEventsFromContext(ctx)`. Optional test helpers live in `coretest`.

## Repository Structure

```text
app/                  Application container and lifecycle
cache/                Redis and Memcached helpers
config/               Environment configuration loading and validation
database/             SQL database initialization
dbtx/                 SQL transaction propagation helpers
docs/                 Architecture, operations, reliability, and release docs
errors/               Application error contract and transport mapping
examples/             Focused integration examples
httpclient/           Resilient outbound HTTP client
grpcclient/           Identity-aware outbound gRPC client
coretest/             Test-only recording logger, fake publisher, and HTTP fixtures
logger/               Structured logging contracts and redaction
messaging/            Kafka abstractions and outbox support
migration/            Goose migration helpers
observability/        Metrics, tracing, request ID, transaction ID
resilience/           Timeout, retry, and circuit breaker helpers
scripts/              Quality, smoke, load, and failure-drill scripts
security/             Auth metadata and JWT verification helpers
server/               gRPC, gateway, and startup orchestration
templates/            Reference package templates
version/              Build/version metadata
```

## Engineering Principles

- Prefer pragmatic, idiomatic Go over framework-heavy abstractions.
- Keep repository behavior as the source of truth; docs should describe actual implementation.
- Make boundaries explicit between transport, service/domain logic, repositories, and infrastructure.
- Keep operational behavior readable in code, logs, metrics, and release evidence.
- Add abstractions only when they remove real duplication or clarify ownership.
- Treat validation honestly: fail fast for required runtime dependencies and surface actionable configuration errors.
- Avoid hidden magic, speculative rewrites, and product-specific business logic in the foundation.

## Documentation

Useful starting points:

- `docs/ARCHITECTURE.md`
- `docs/SERVICE_BOOTSTRAP.md`
- `docs/CONFIGURATION_PROFILES.md`
- `docs/ERROR_HANDLING.md`
- `docs/OBSERVABILITY.md`
- `docs/MESSAGING_PATTERN.md`
- `docs/RELIABILITY.md`
- `docs/SECURITY.md`
- `docs/TECHNICAL_GUIDE.md`
- `docs/DATA_MODEL.md`
- `docs/VERSIONING.md`
- `MIGRATION.md`

Architecture diagram placeholder:

```text
Client -> Gateway/gRPC -> Handler -> Service -> Repository -> Database
                         |          |          |
                         |          |          +-> Outbox/Kafka
                         |          +-> Logger/Metrics/Tracing
                         +-> Middleware/Interceptors/Auth/Recovery
```

## Release Discipline

CI should remain the fast baseline:

```bash
make test
make vet
make lint
```

Release candidates should use the stronger local/release checks where applicable:

```bash
make quality-gate
BASE_URL=https://staging.example.com make smoke-gate
BASE_URL=https://staging.example.com TARGET_PATH=/health make load-steady
```

Use `docs/PRODUCTION_SIGNOFF.md` and `docs/RELEASE_EVIDENCE_TEMPLATE.md` when preparing a production service release that consumes this module.

Open release work remains: internal module path (`D-PATH`) and human-controlled release/tagging. `Q-SIG` confirmation is deferred but is not a release blocker. GC-23 uses OpenTelemetry OTLP to Elastic APM Server; GC-24 metrics are opt-in via `METRICS_ENABLED` (default false). The source directive stays Go 1.24.3; the office image uses 1.24.13 and the next upgrade target is 1.26.x.

Set build metadata with `-ldflags` when building a service that exposes `/version`:

```bash
go build -ldflags "\
  -X 'github.com/yogayulanda/go-core/version.Version=1.0.0' \
  -X 'github.com/yogayulanda/go-core/version.Commit=$(git rev-parse HEAD)' \
  -X 'github.com/yogayulanda/go-core/version.BuildDate=$(date -u +%Y-%m-%dT%H:%M:%SZ)'"
```

## Roadmap

- Keep compatibility and migration guidance clear for public releases.
- Continue tightening docs around service bootstrap and operational behavior.
- Expand examples only when they reflect real implementation patterns.
- Avoid adding product-specific assumptions to the foundation.

## Contributing

See `CONTRIBUTING.md`.

## Security

See `SECURITY.md` for responsible disclosure guidance.

## License

Apache License 2.0. See `LICENSE`.
