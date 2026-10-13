# Configuration Profiles

Configuration is loaded by `config.Load` and validated by `cfg.Validate()`. Runtime tuning values are code fields unless listed as environment variables below.

## Required Service Profile

- `SERVICE_NAME` (required)
- `APP_ENV` (required: `local`, `dev`, `sit`, `uat`, or `prod`)
- `LOG_LEVEL` (default `info`)
- `GRPC_PORT` (default `50051`)
- `HTTP_PORT` (default `8080`)

## HTTP Profile

- `HTTP_CORS_ALLOWED_ORIGINS` (empty disables CORS)
- `HTTP_PPROF_ENABLED` (default false; pprof is rejected in production)
- `AUTH_SIGNATURE_MASTER_KEY` (non-empty enables signature validation; scheme confirmation remains `Q-SIG`)

HTTP timeouts, maximum body size, CORS allowed headers, and trusted proxy hops are code fields under `config.HTTPConfig`. Defaults include read-header 5s, read 15s, write 60s, idle 120s, body limit 1 MiB, and trusted proxy hops 0.

## Database Profile

- `DB_LIST`
- `DB_<ALIAS>_DRIVER`: `sqlserver` or `mysql`
- `DB_<ALIAS>_HOST`, `PORT`, `NAME`, `USER`, `PASSWORD`
- `DB_<ALIAS>_PARAMS` (optional)

Pool sizing and required/optional status are code fields. PostgreSQL and old pool/TLS env overrides are unsupported and ignored/reported.

## Kafka Profile

- `KAFKA_BROKERS` enables Kafka
- `KAFKA_USERNAME`, `KAFKA_PASSWORD`, `KAFKA_SASL_MECHANISM` (`PLAIN`, `SCRAM-SHA-256`, `SCRAM-SHA-512`)
- `KAFKA_JKS_FILE`, `KAFKA_JKS_PASSWORD`

`KAFKA_JKS_FILE` autodetects JKS and PEM trust material. Credentials and TLS are shared by producer, consumer, and readiness probes. The office's selected mechanism is still an operations question; all approved mechanisms are supported.

## Observability And Diagnostics

- `APM_ENDPOINT` enables the existing OTLP gRPC tracing exporter.
- `METRICS_ENABLED` enables metric recording and `/metrics` (default false).
- `DIAG_ENABLED` controls diagnostic snapshots (default true).

Tracing goes to Elastic APM Server via OpenTelemetry OTLP. No Elastic APM agent is used and logs do not contain `transaction.id`.
Diagnostics thresholds, cooldown (10m), and check interval (30s) are code fields.

## Code-Only Runtime Fields

- `AppConfig`: `DrainDelay` 3s, `ShutdownTimeout` 25s.
- `HTTPConfig`: timeouts, body limit, CORS headers, trusted proxy hops.
- `DiagnosticsConfig`: heap threshold 80%, goroutine threshold 5000, cooldown 10m, check interval 30s.
- `GRPCConfig` / `HTTPConfig`: server TLS fields; defaults are off.
- DB pool limits, migration options, logging thresholds, trace sampling, and cache details.
