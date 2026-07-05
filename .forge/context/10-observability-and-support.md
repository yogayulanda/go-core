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
- Metrics include HTTP/gRPC request metrics, additive service operation metrics, DB operation metrics, messaging publish/consume metrics, outbox batch metrics, and optional transaction counters.
- Tracing is optional and based on OpenTelemetry; context propagation is expected across layers.
- Sensitive data should be redacted from logs.

## Assumptions
- Concrete dashboards, alert routing, and SLO thresholds beyond the release/load gates are service-environment responsibilities.

## Related Files
- `09-errors-and-resilience.md`
- `11-testing-and-quality.md`
- `13-operations-and-runbook.md`
