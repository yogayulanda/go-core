# Proposed Context Patch: 14 Decisions And Constraints

Status: promoted on 2026-10-14 to `.forge/context/14-decisions-assumptions-and-constraints.md`.

## Record accepted implementation decisions
- HTTP retries are per-client opt-in; returned HTTP statuses are responses; body logs require caller masker; HTTP identity forwarding requires `WithInternalPropagation`.
- gRPC default transport is plaintext to match the server default; callers configure TLS credentials explicitly.
- JKS reader `keystore-go/v4 v4.5.0` is accepted as low-churn for stable JKS format. It is isolated in `messaging/internal/kafkatls`, PEM is autodetected, MIT attribution is in `NOTICE.txt`, and no govulncheck finding targeted it.
- Outbox claims per key with leases, publishes outside claim transactions, bounds retry attempts, and `DEAD` blocks later events for that key.
- Diagnostics use `runtime.Stack` detail level 1 to avoid argument disclosure. The 10 ms GC-pause upper-bound limit is an assertion in the 5000-goroutine benchmark, not a runtime guard.

## Preserve open constraints
- `GC-23` is decided as OTLP to Elastic APM Server without an Elastic APM agent or `transaction.id`.
- `GC-24` is decided: `METRICS_ENABLED` defaults false, metrics and `/metrics` are conditional, and `app_transaction_total` is removed.
- `GC-27` accepts caller-controlled HTTP identity propagation without host validation and Kafka identity restoration trusted on the internal network without per-message identity authentication.
- `D-PATH` is deferred until a downstream fork chooses its module path; this public release keeps the repository's current path. `Q-SIG` is deferred but is not a release blocker.
- Keep Go directive 1.24.3, use Go 1.24.13 for the current build image, and target Go 1.26.x next.
- The deployment's Kafka SASL preference and termination grace-period configuration remain operational confirmations.

Evidence: WS-5/6/7a implementation commits, `NOTICE.txt`, SDD revision 9 (2026-10-14).
