# Proposed Context Patch: 04 Interfaces And Contracts

Status: promoted on 2026-10-14 to `.forge/context/04-interfaces-and-contracts.md`.

## Replace stale confirmed facts
- Replace the prior error code set/`ErrorResponse` description with an open `errors.Def` registry, `errors.AppError`, and `response` gateway envelope.
- Replace the old envelope field claim with the implemented fields: `status`, `response_code`, `message`, `request_id`, `timestamp`; success adds `data`, validation failure may add `details`.
- Clarify `messaging.Message` includes topic, key, payload, headers, optional `EventID`; `messaging.CloudEventAttributes` is available via `CloudEventsFromContext`.

## Add confirmed facts
- HTTP transport failures use `httpclient.TransportError` with `before_send`/`after_send` phase and kind. HTTP responses, including 5xx, return a response and nil error.
- HTTP retry is opt-in; internal identity propagation requires `WithInternalPropagation()`.
- `grpcclient.NewClient` adds a 30-second default deadline only when the caller context has no deadline, identity propagation, OTel stats, call logs, keepalive, and disabled service-config retry. Client metrics are disabled unless `grpcclient.WithMetricsEnabled(true)` is supplied.
- A successful REST request reaching gRPC has one access log from the gRPC interceptor; gateway middleware records metrics but no second access log.
- CloudEvents audit outbox publication requires UUID `ce_id`, `ce_subject`, and `ce_sequence`.

Evidence: `errors/codes.go`, `server/gateway/envelope_middleware.go`, `httpclient/transport_error.go`, `grpcclient/client.go`, `messaging/message.go`, `messaging/outbox/audit.go`, `logger/service.go`.
