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
- Service, database, messaging, outbox and gRPC request metric families are exposed on `/metrics`.
- `app_transaction_total` remains registered but is not incremented by `LogTransaction`; removal versus implementation remains part of blocked `GC-24` policy work.

## Tracing

Tracing uses the existing OpenTelemetry SDK and OTLP gRPC exporter, enabled by `APM_ENDPOINT`. The choice between OTLP-to-Elastic APM and an Elastic APM Go agent is unresolved (`GC-23`); do not infer that `transaction.id` is emitted.

## Runtime Diagnostics

`DIAG_ENABLED` defaults to true. The watcher performs periodic runtime-metric and goroutine-count checks. It emits one `diagnostic_snapshot` warning when the goroutine threshold or cgroup heap threshold is reached, subject to a shared cooldown. If cgroup memory is unavailable or unlimited, heap triggering is disabled and reported once; the goroutine trigger remains active. Snapshots contain bounded function/file locations and numeric stats, never goroutine arguments.

Defaults: heap threshold 80%, goroutine threshold 5000, cooldown 10 minutes, check interval 30 seconds.

## Open Policy Decisions

- `GC-23`: tracing backend/agent choice.
- `GC-24`: metrics enablement default and transaction counter policy.
