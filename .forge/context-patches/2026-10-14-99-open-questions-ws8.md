# Proposed Context Patch: 99 Open Questions

Status: promoted on 2026-10-14 to `.forge/context/99-open-questions.md`.

## Keep open
- `D-PATH`: choose an import path if this repository is later used as a downstream fork; the public v1.1.0 release keeps the current path.
- `Q-SIG`: compare the existing webview/service signature scheme before changing or certifying the contract; it is not a release blocker.
- Operations: confirm the deployment's Kafka SASL preference and termination grace period.
- S8: run SQL Server/MySQL outbox integration tests when Docker/test databases are available.

## Resolved
- `GC-23`: OpenTelemetry OTLP to Elastic APM Server via `APM_ENDPOINT`; no Elastic agent and no `transaction.id`.
- `GC-24`: `METRICS_ENABLED=false` by default; recording and `/metrics` are conditional; remove `app_transaction_total`.
- Keep `go 1.24.3`; current build image Go 1.24.13; next target Go 1.26.x.

## Close stale questions only with evidence
- The implementation supports PLAIN and both SCRAM variants regardless of the still-unknown deployment preference.
- JKS and PEM trust material are supported; the JKS parser decision and dependency version are recorded in active context `.forge/context/14-decisions-assumptions-and-constraints.md`.
- Diagnostics snapshot pause measured 0.229ms upper bound in the recorded 5000-goroutine benchmark, below the 10ms stop limit.
