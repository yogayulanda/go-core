# Proposed Context Patch: 14 Decisions And Constraints

Status: proposal only; curated context remains unchanged.

## Record accepted implementation decisions
- HTTP retries are per-client opt-in; returned HTTP statuses are responses; body logs require caller masker; HTTP identity forwarding requires `WithInternalPropagation`.
- gRPC default transport is plaintext to match the server default; callers configure TLS credentials explicitly.
- JKS reader `keystore-go/v4 v4.5.0` is accepted as low-churn for stable JKS format. It is isolated in `messaging/internal/kafkatls`, PEM is autodetected, MIT attribution is in `NOTICE.txt`, and no govulncheck finding targeted it.
- Outbox claims per key with leases, publishes outside claim transactions, bounds retry attempts, and `DEAD` blocks later events for that key.
- Diagnostics use `runtime.Stack` detail level 1 to avoid argument disclosure; snapshots stop if GC pause upper bound exceeds 10ms.
- No `.forge/context` file is directly edited; these are proposals only.

## Preserve open constraints
- `GC-23`, `GC-24`, `D-PATH`, and `Q-SIG` remain unresolved; no module path change or tag is made.
- Office Kafka SASL preference and production termination grace-period configuration remain operational confirmations.

Evidence: WS-5/6/7a implementation commits, `NOTICE.txt`, SDD revision 8.
