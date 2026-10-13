# Proposed Context Patch: 09 Errors And Resilience

Status: proposal only; curated context remains unchanged.

## Replace stale confirmed facts
- The current error API is registry-backed with open codes and built-ins under `CORE-*`; application helpers are `errors.New`, `Wrap`, and `Validation`. Prior builder/category/domain APIs are absent.
- HTTP client retries are disabled by default; a received status is not a transport error. `TransportError` classifies phase/kind.
- HTTP 413 uses `CORE-1005`; validation remains `CORE-1000` / HTTP 400.

## Add confirmed facts
- Unknown response codes resolve to `CORE-9000`; gateway exposes structured details only for validation.
- Readiness returns 503 during drain; gRPC health reports `NOT_SERVING`.
- Outbox retries are bounded and per event, use lease state/backoff, and transition to `DEAD` at the configured attempt limit. `DEAD` blocks later rows with the same key.

Evidence: `errors/codes.go`, `errors/error.go`, `httpclient/client.go`, `httpclient/transport_error.go`, `server/gateway/error_handler.go`, `app/runtime_state.go`, `messaging/outbox/worker.go`.
