# Configuration Profiles

Use this document to reason about configuration in groups instead of one long env list.

## 1. Core Required Profile

Always decide first:

- `SERVICE_NAME`
- `APP_ENV`
- `LOG_LEVEL`
- `SHUTDOWN_TIMEOUT`
- `GRPC_PORT`
- `HTTP_PORT`

## 2. Transport Security Profile

Enable only when the service needs transport TLS:

- `GRPC_TLS_*`
- `HTTP_TLS_*`

## 3. Database Profile

Enable only when the service needs SQL storage:

- `DB_LIST`
- `DB_<NAME>_*`

Notes:

- required databases fail fast on startup and affect `/ready`
- optional databases do not stop startup but still appear in readiness checks

## 4. Migration Profile

Enable only when the service chooses startup migration:

- `MIGRATION_AUTO_RUN`
- `MIGRATION_DB`
- `MIGRATION_DIR`
- `MIGRATION_LOCK_*`

## 5. Observability Profile

Optional tracing and stronger telemetry:

- `OTEL_EXPORTER_OTLP_ENDPOINT`
- `OTEL_EXPORTER_OTLP_INSECURE`
- `OTEL_EXPORTER_OTLP_CA_CERT_FILE`
- `TRACE_SAMPLING_RATIO`

## 6. Infra Dependency Profiles

Enable per dependency:

- Redis: `REDIS_*`
- Memcached: `MEMCACHED_*` (plus legacy `MEMCACHE_HOST/PORT` fallback)
- Kafka: `KAFKA_*` (supports SASL Plain and JKS certificates)

Cache notes:

- `REDIS_ENABLED=true` means Redis is part of the chosen runtime dependency set
- `MEMCACHED_ENABLED=true` means Memcached is part of the chosen runtime dependency set
- enabled caches fail fast during startup if the dependency is unavailable
- enabled caches are treated as required by `/ready`
- Memcached readiness intentionally treats cache miss as healthy so the probe stays lightweight and deterministic

## 7. Auth Profile

Enable internal JWT verification only when required:

- `INTERNAL_JWT_*`

## Validation DX

Use:

- `cfg.Validate()` for the simple public error path
- `cfg.ValidateIssues()` when the caller wants structured validation issues grouped by section and field
