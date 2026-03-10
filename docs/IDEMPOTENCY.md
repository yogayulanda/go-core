# Idempotency

Service-owned concern; `go-core` provides helpers only.

Recommended record:
- idempotency key
- request hash
- status/result snapshot
- expiry timestamp

Same key + different payload should be rejected.
