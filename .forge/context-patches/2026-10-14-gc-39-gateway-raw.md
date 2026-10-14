# Proposed Context Patch: GC-39 Gateway Raw Responses

Status: promoted on 2026-10-14 to `.forge/context/04-interfaces-and-contracts.md`.

## Decision

- `gateway.Raw(runtime.HandlerFunc)` and `gateway.RawHTTP(http.Handler)` explicitly opt a registered handler out of the gateway's success envelope.
- The opt-out is request-scoped and set through a mutable marker carried by the request context; it does not depend on path names or response-writer type assertions.
- Raw preserves the handler's status, headers, and body at the envelope boundary. Other gateway middleware remains active, including request identity, body limits, signature validation, panic recovery, tracing, CORS, metrics, and access logging.
- The handler owns the complete response contract, including success and error formats. Use Raw for transparent proxies, webhooks, and downloads, not ordinary APIs that rely on the standard envelope.
- gRPC-backed routes emit the standard access log from the gRPC logging interceptor. Raw emits one access log through `logger.LogService` with standard ECS/access fields and request identity; request/response bodies and arbitrary headers are not logged.
- Successful direct `runtime.ServeMux.HandlePath` handlers that do not enter gRPC have no access log today. This behavior is intentionally unchanged for v1.2.0 and tracked in `docs/TODO_BACKLOG.md`.

## Evidence

- `server/gateway/raw.go` exports both adapters.
- `server/gateway/envelope_middleware.go` applies the request-scoped marker and preserves raw responses.
- `server/gateway/raw_test.go` exercises raw and enveloped responses through the real gateway chain, including panic and body-limit paths.
- `server/gateway/raw_test.go` also verifies the Raw access-log schema and absence of request/response body and arbitrary header values.
- `server/grpc/interceptor.go` contains the gRPC access-log call; `logger/service.go` applies the shared access schema and `logger/zap.go` adds ECS/context identity fields.
- `docs/TECHNICAL_GUIDE.md` documents consumer use and responsibility.

Do not promote this proposal without maintainer review.