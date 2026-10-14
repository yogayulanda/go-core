# Decisions Assumptions and Constraints

## When to Read
- Read before broadening repository scope, adding new shared contracts, or changing release-sensitive behavior.

## Do Not Use This For
- Open unknowns: `99-open-questions.md`.
- Stable term definitions: `98-glossary.md`.

## Source of Truth
- Cross-cutting repository decisions, durable assumptions, and constraints that shape future implementation work.

## Current Context
- This file holds cross-cutting rules that apply across packages and docs.

## Confirmed Facts
- `go-core` is constrained to reusable foundation/runtime behavior and explicitly excludes business entities, business rules, and service-specific workflow logic.
- Optional infrastructure must remain explicit and service-controlled; the foundation should not hide background behavior such as outbox worker startup or optional dependency activation.
- Public-contract changes must update README/docs/tests and `MIGRATION.md` when upgrade behavior changes.
- Release discipline assumes semantic versioning from `v1.0.0` onward and treats tagged releases as the adoption source of truth.
- Current auth config validation requires `INTERNAL_JWT_PUBLIC_KEY` whenever `INTERNAL_JWT_ENABLED=true`, even when `INTERNAL_JWT_JWKS_ENDPOINT` is also set.
- `GC-23` is resolved to OpenTelemetry OTLP to Elastic APM Server via `APM_ENDPOINT`; no Elastic APM agent or `transaction.id` log field is used.
- `GC-24` is resolved: `METRICS_ENABLED` defaults false, metric recording is gated, `/metrics` returns 404 while disabled, and `app_transaction_total` is removed.
- `GC-27` accepts the identity trust boundary: HTTP propagation is caller-controlled and does not validate destination hosts; Kafka identity headers are trusted based on internal network/producer controls, without per-message identity authentication.
- `GC-28`: Follow a rolling N-1 Go toolchain policy: use the latest patch of the N-1 minor line, advance the minor target with Go's February/August releases, and apply security patches within 14 days. Set the `go` directive to the minimum required by the selected dependency graph as determined by `go mod tidy`; do not set it to the installed patch version. An official maintainer's minimum security fix may be younger than 14 days when required to remediate an advisory; record its release date, age, and exception.
- The current dependency graph requires `go 1.26.0`; CI and release validation use Go 1.26.9.
- Go 1.27 is outside the current v1.2.0 target.
- `D-PATH` is deferred until a downstream fork; the public repository release does not change the module path.
- `Q-SIG` confirmation remains deferred but is not a release blocker.

## Assumptions
- Consuming services own deployment topology, secret management, authorization policy, and business payload compatibility.

## Related Files
- `03-domain-boundaries.md`
- `11-testing-and-quality.md`
- `99-open-questions.md`
