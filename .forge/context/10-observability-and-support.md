# Observability and Support

## When to Read
- Read before changing structured logs, metrics, tracing hooks, or runtime support signals.

## Do Not Use This For
- Release/runbook procedures: `13-operations-and-runbook.md`.
- Env var and enablement details: `12-runtime-deployment-and-config.md`.

## Source of Truth
- Stable observability baseline emitted by the foundation and referenced by adopting services/operators.

## Current Context
- The repo defines a shared observability baseline rather than a service-specific dashboard or alert catalog.

## Confirmed Facts
- Structured logging contracts include `logger.ServiceLog`, `logger.DBLog`, and the platform-standard `logger.TransactionLog` for transaction-oriented services only.
- Runtime orchestration logs include `app_init`, `app_runtime`, `lifecycle_shutdown`, `shutdown_hook`, `runtime_orchestration`, `component_start`, and startup readiness events.
- Transport observability emits `grpc_request` and `http_request` service logs plus request and service metrics.
- Messaging observability emits `message_publish`, `message_consume`, `outbox_worker`, and `outbox_batch` service logs plus publish/consume/process/outbox metric families.
- Metrics include HTTP/gRPC request metrics, service operation metrics, DB operation metrics, messaging publish/consume metrics, and outbox batch metrics. Recording defaults off and requires `METRICS_ENABLED=true`; the gateway always registers `/metrics` but returns 404 while disabled. `app_transaction_total` is removed.
- Tracing uses the OpenTelemetry SDK and OTLP gRPC exporter to Elastic APM Server via `APM_ENDPOINT`; no Elastic APM agent is used, and logs do not include `transaction.id`.
- `DIAG_ENABLED` defaults true. Diagnostics check every 30 seconds, trigger at 80% cgroup heap use or 5000 goroutines, and share a 10-minute cooldown. The 10 ms GC-pause upper bound is asserted by the 5000-goroutine benchmark only; it is not a runtime snapshot guard.
- Sensitive data should be redacted from logs.

## Assumptions
- Concrete dashboards, alert routing, and SLO thresholds beyond the release/load gates are service-environment responsibilities.

## Related Files
- `09-errors-and-resilience.md`
- `11-testing-and-quality.md`
- `13-operations-and-runbook.md`
