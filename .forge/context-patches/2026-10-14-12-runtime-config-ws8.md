# Proposed Context Patch: 12 Runtime Deployment And Config

Status: proposal only; curated context remains unchanged.

## Replace stale confirmed facts
- `SERVICE_NAME` and `APP_ENV` are required. `LOG_LEVEL`, `GRPC_PORT`, `HTTP_PORT`, Kafka SASL/JKS settings, `APM_ENDPOINT`, `DIAG_ENABLED`, and `HTTP_PPROF_ENABLED` are the current relevant env variables; no `SHUTDOWN_TIMEOUT`, `OTEL_EXPORTER_OTLP_ENDPOINT`, `REDIS_ENABLED`, or `MEMCACHED_ENABLED` env override is used.
- Drain/shutdown timeout, gateway timeouts/body limit/proxy hops, diagnostics thresholds, logger thresholds, and DB pool configuration are code fields.

## Add confirmed facts
- Defaults: `DrainDelay=3s`, `ShutdownTimeout=25s`; `ReadHeaderTimeout=5s`, `ReadTimeout=15s`, `WriteTimeout=60s`, `IdleTimeout=120s`, body limit 1MiB; diagnostics 80% heap, 5000 goroutines, 10m cooldown, 30s interval.
- `KAFKA_JKS_FILE` supports JKS or PEM; SASL mechanisms are PLAIN/SCRAM-SHA-256/SCRAM-SHA-512.
- During drain, `/ready` is 503 and gRPC health is NOT_SERVING.

Evidence: `config/load.go`, `config/config.go`, `config/validate.go`, `.env.example`, `app/app.go`, `server/gateway/health.go`, `server/grpc/server.go`.
