# Observability

This document describes the implemented observability behavior. Dashboard/backend policy decisions still open in the SDD are identified separately.

## Logs

- ECS JSON is used outside `APP_ENV=local`; local uses a console encoder.
- Context identity is injected as `labels.request_id`, `labels.gtid`, `user.id`, `client.ip`, and `user_agent.original` when present.
- Service/database/event/transaction log-contract fields use `app.*`; event identity uses ECS `event.*`.
- A normal REST request that reaches the local gRPC service produces exactly one access log from the gRPC interceptor. The gateway HTTP metrics middleware records metrics only. Gateway access logs are reserved for failures before gRPC (for example body-limit, signature, panic, or routing failures).
- `/health`, `/ready`, `/metrics`, `/version`, and gRPC health checks are not logged as access requests unless they fail.

## Metrics

- `app_http_request_total{service,method,route,status}` and `app_http_request_duration_seconds{service,method,route}`.
- The `route` label comes from the matched grpc-gateway `runtime.Pattern`; unmatched paths use the constant `unmatched`. Raw request paths are not used as labels.
- Metrics are recorded and `/metrics` serves scrape data only when `METRICS_ENABLED=true`; default is false. Disabled requests receive 404.
- When enabled, service, database, messaging, outbox and gRPC request metric families are exposed on `/metrics`.
- `app_transaction_total` has been removed.

## Tracing

Tracing uses the existing OpenTelemetry SDK and OTLP gRPC exporter to Elastic APM Server, enabled by `APM_ENDPOINT`. No Elastic APM Go agent is used. `transaction.id` does not appear in logs.

## Runtime Diagnostics

`DIAG_ENABLED` defaults to true. The watcher performs periodic runtime-metric and goroutine-count checks. It emits one `diagnostic_snapshot` warning when the goroutine threshold or cgroup heap threshold is reached, subject to a shared cooldown. If cgroup memory is unavailable or unlimited, heap triggering is disabled and reported once; the goroutine trigger remains active. Snapshots contain bounded function/file locations and numeric stats, never goroutine arguments.

Defaults: heap threshold 80%, goroutine threshold 5000, cooldown 10 minutes, check interval 30 seconds.
