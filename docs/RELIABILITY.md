# Reliability

- Fail fast for required dependencies.
- Allow graceful degradation for optional dependencies.
- `/ready` returns `503` when required dependency is down.
- Enabled Redis and Memcached are treated as required dependencies for startup and `/ready`.
- Cache startup should emit explicit runtime signals so dependency failures are diagnosable without exposing raw internals to clients.
- Memcached health intentionally treats cache miss as healthy; connection and timeout failures still mark readiness down.
- Use bounded timeout/retry for outbound calls.
- Auto migration should use lock to avoid concurrent runners.
- Prefer logger-aware resilience hooks when retry and timeout behavior needs explicit runtime diagnostics.
- Prefer `migration.AutoRunUpWithLogger(...)` when startup migration behavior should be visible in service runtime logs.
