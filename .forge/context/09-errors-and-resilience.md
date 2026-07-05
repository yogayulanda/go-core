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
- `errors.AppError` carries code, message, user message, category, finality, retryability, details, and optional wrapped internal error.
- Validation errors may include structured `details`; unknown or internal errors must be sanitized before leaving the service boundary.
- Readiness rules are strict for enabled required dependencies: `/ready` returns `503` when a required dependency is unavailable, while optional databases may be skipped during startup.
- Enabled Redis and Memcached are treated as required dependencies for startup and readiness, even though enabling them is optional at the config level.
- Reliability guidance calls for bounded timeout/retry on outbound calls, migration locking for startup migrations, and logger-aware resilience hooks when runtime diagnostics matter.
- `server.Run(...)` ignores expected server-stop errors, but returns joined failures for real component errors or shutdown wait timeouts.

## Assumptions
- None.

## Related Files
- `07-integrations-and-dependencies.md`
- `08-security-and-access.md`
- `10-observability-and-support.md`
