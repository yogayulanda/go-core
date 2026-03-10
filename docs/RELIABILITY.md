# Reliability

- Fail fast for required dependencies.
- Allow graceful degradation for optional dependencies.
- `/ready` returns `503` when required dependency is down.
- Use bounded timeout/retry for outbound calls.
- Auto migration should use lock to avoid concurrent runners.
