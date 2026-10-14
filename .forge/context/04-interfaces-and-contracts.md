# Interfaces and Contracts

## When to Read
- Read before changing exported transport behavior, event envelopes, or public error/auth contracts.

## Do Not Use This For
- Dependency ownership and provider notes: `07-integrations-and-dependencies.md`.
- Secret handling and auth policy detail: `08-security-and-access.md`.

## Source of Truth
- Transport-facing contracts and compatibility-relevant technical interfaces exposed by the foundation.

## Current Context
- Consuming services own their protobuf, HTTP route, and business payload schemas; `go-core` owns reusable transport behavior and technical envelopes.

## Confirmed Facts
- gRPC server construction is provided by `server/grpc.New(...)`, which installs recovery, request ID, auth, logging, and metrics interceptors plus OpenTelemetry gRPC stats handling. Metric recording is controlled by `METRICS_ENABLED` and defaults off.
- The HTTP gateway created by `server/gateway.New(...)` always registers `/health`, `/ready`, `/version`, and `/metrics`; `/metrics` returns 404 when `METRICS_ENABLED` is false. Pprof endpoints are conditional on `HTTP.PprofEnabled` and forbidden in production.
- Gateway success envelopes contain `status`, `response_code`, `message`, `request_id`, `timestamp`, and `data`; validation failures may include `details`. Application errors use the open `errors.Def` registry and `errors.AppError`.
- `gateway.Raw(runtime.HandlerFunc)` and `gateway.RawHTTP(http.Handler)` opt an explicitly registered route out of the success envelope using a request-context marker. Raw handlers retain the other gateway middleware and own the full response contract: status, headers, body, and error format.
- gRPC-backed routes receive their standard access log from the gRPC logging interceptor. Raw gateway routes emit one standard access log through `logger.LogService`; it contains ECS/access fields and request identity, not request/response bodies or arbitrary headers. Direct successful non-Raw `runtime.ServeMux.HandlePath` handlers do not currently emit an access log; this remains a deferred follow-up.
- `messaging.Message` includes topic, key, payload, headers, and optional `EventID`. CloudEvent attributes can be retrieved from context with `messaging.CloudEventsFromContext`.
- HTTP transport failures use `httpclient.TransportError` with phase and kind. Received HTTP statuses, including 5xx, return a response with nil error. Retries are opt-in; identity propagation requires caller-supplied `WithInternalPropagation()`.
- `grpcclient.NewClient` adds a 30-second default deadline only when the caller context has none, plus identity propagation, OpenTelemetry stats, call logs, and keepalive. Service-config retries and metrics are disabled by default; metrics require `grpcclient.WithMetricsEnabled(true)`.
- CloudEvents audit outbox publication requires a UUID `ce_id`, `ce_subject`, and `ce_sequence`.

## Assumptions
- None.

## Related Files
- `07-integrations-and-dependencies.md`
- `08-security-and-access.md`
- `09-errors-and-resilience.md`
