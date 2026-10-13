# Proposed Context Patch: 10 Observability And Support

Status: proposal only; curated context remains unchanged.

## Replace stale confirmed facts
- Normal REST requests that reach gRPC produce exactly one access log from the gRPC interceptor. Gateway middleware adds metrics only; gateway logs are for pre-gRPC failures.
- HTTP `route` metrics use the grpc-gateway matched `runtime.Pattern.String()`; unmatched paths use the constant `unmatched`.

## Add confirmed facts
- Logs use ECS JSON except `APP_ENV=local`, which uses the console encoder. Context identity fields include `labels.request_id`, `labels.gtid`, `user.id`, `client.ip`, and `user_agent.original`.
- `DIAG_ENABLED` defaults true; diagnostics run at 30s intervals, trigger at 80% cgroup heap or 5000 goroutines, share a 10m cooldown, and omit local variable/argument values.
- Benchmarks observed: periodic check 27.3µs; 5000-goroutine snapshot 20.13ms total with 0.229ms GC pause upper bound.
- Prometheus metrics are recorded and `/metrics` is registered only when `METRICS_ENABLED=true`; default is false.
- `app_transaction_total` has been removed.
- Tracing is OpenTelemetry OTLP to Elastic APM Server via `APM_ENDPOINT`; there is no Elastic APM agent and logs do not include `transaction.id`.

## Preserve toolchain direction
- Keep the source directive at Go 1.24.3; office build image is 1.24.13; next upgrade target is Go 1.26.x.

Evidence: `logger/zap.go`, `server/gateway/http_metrics_middleware.go`, `observability/diagnostics/diagnostics.go`, benchmark output, `observability/metrics.go`.
