# Errors and Resilience

## When to Read
- Read before changing canonical errors, retry/timeout behavior, readiness semantics, or graceful-stop behavior.

## Do Not Use This For
- Metrics and logging inventory: `10-observability-and-support.md`.
- Provider list: `07-integrations-and-dependencies.md`.

## Source of Truth
- Error taxonomy, retry/failure guidance, dependency degradation rules, and runtime shutdown behavior.

## Current Context
- The repo separates compact public transport errors from richer internal observability signals.

## Confirmed Facts
- The current error API uses an open `errors.Def` registry and `errors.AppError`; application helpers include `errors.New`, `Wrap`, and `Validation`. Built-in codes use the `CORE-*` namespace.
- Unknown response codes resolve to `CORE-9000`; gateway responses expose structured details only for validation errors. HTTP 413 maps to `CORE-1005`; validation maps to `CORE-1000` / HTTP 400.
- HTTP client retries are disabled by default and opt-in per client. A received status, including 5xx, is a response rather than a transport error; `httpclient.TransportError` classifies transport phase and kind.
- Readiness rules are strict for enabled required dependencies: `/ready` returns `503` when a required dependency is unavailable, while optional databases may be skipped during startup.
- Enabled Redis and Memcached are treated as required dependencies for startup and readiness, even though enabling them is optional at the config level.
- Reliability guidance calls for bounded timeout/retry on outbound calls, migration locking for startup migrations, and logger-aware resilience hooks when runtime diagnostics matter.
- `server.Run(...)` ignores expected server-stop errors, but returns joined failures for real component errors or shutdown wait timeouts.
- During drain, `/ready` returns 503 and gRPC health reports `NOT_SERVING`.
- Outbox retries are bounded per event, use lease state and backoff, and transition to `DEAD` at the configured attempt limit. An earlier `DEAD` row blocks later rows for the same key.

## Assumptions
- None.

## Related Files
- `07-integrations-and-dependencies.md`
- `08-security-and-access.md`
- `10-observability-and-support.md`
