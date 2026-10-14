# Proposed Context Patch: 10 Observability And Support

Status: promoted on 2026-10-14 to `.forge/context/10-observability-and-support.md`.

## Replace stale confirmed facts
- Normal REST requests that reach gRPC produce exactly one access log from the gRPC interceptor. Gateway middleware adds metrics only; gateway logs are for pre-gRPC failures.
- HTTP `route` metrics use the grpc-gateway matched `runtime.Pattern.String()`; unmatched paths use the constant `unmatched`.

## Add confirmed facts
- Logs use ECS JSON except `APP_ENV=local`, which uses the console encoder. Context identity fields include `labels.request_id`, `labels.gtid`, `user.id`, `client.ip`, and `user_agent.original`.
- `DIAG_ENABLED` defaults true; diagnostics run at 30s intervals, trigger at 80% cgroup heap or 5000 goroutines, share a 10m cooldown, and omit local variable/argument values.
- The 5000-goroutine diagnostic benchmark asserts that its measured GC-pause upper bound stays at or below 10 ms; this is a benchmark assertion, not a runtime snapshot guard.
- `METRICS_ENABLED` defaults to false. The gateway always registers `/metrics`, returns 404 while disabled, and serves scrape data only when enabled; metric recording is also gated.
- `app_transaction_total` has been removed.
- Tracing is OpenTelemetry OTLP to Elastic APM Server via `APM_ENDPOINT`; there is no Elastic APM agent and logs do not include `transaction.id`.

## Preserve toolchain direction
- Keep the source directive at Go 1.24.3; current build image is Go 1.24.13; next upgrade target is Go 1.26.x.

Evidence: `logger/zap.go`, `server/gateway/http_metrics_middleware.go`, `observability/diagnostics/diagnostics.go`, benchmark output, `observability/metrics.go`.
