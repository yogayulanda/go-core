# Proposed Context Patch: GC-39 Gateway Raw Responses

Status: proposed; review required; not promoted.

## Decision

- `gateway.Raw(runtime.HandlerFunc)` and `gateway.RawHTTP(http.Handler)` explicitly opt a registered handler out of the gateway's success envelope.
- The opt-out is request-scoped and set through a mutable marker carried by the request context; it does not depend on path names or response-writer type assertions.
- Raw preserves the handler's status, headers, and body at the envelope boundary. Other gateway middleware remains active, including request identity, body limits, signature validation, panic recovery, tracing, CORS, metrics, and access logging.
- The handler owns the complete response contract, including success and error formats. Use Raw for transparent proxies, webhooks, and downloads, not ordinary APIs that rely on the standard envelope.

## Evidence

- `server/gateway/raw.go` exports both adapters.
- `server/gateway/envelope_middleware.go` applies the request-scoped marker and preserves raw responses.
- `server/gateway/raw_test.go` exercises raw and enveloped responses through the real gateway chain, including panic and body-limit paths.
- `docs/TECHNICAL_GUIDE.md` documents consumer use and responsibility.

Do not promote this proposal without maintainer review.