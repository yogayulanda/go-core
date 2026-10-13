# Proposed Context Patch: 99 Open Questions

Status: proposal only; curated context remains unchanged.

## Keep open
- `GC-23`: choose OTLP-to-Elastic APM versus Elastic APM Go agent.
- `GC-24`: decide metrics enablement default and transaction counter handling.
- `D-PATH`: approve the internal module path; WS-9 is excluded and no tag was created.
- `Q-SIG`: compare the existing webview/service signature scheme before changing or certifying the contract.
- Operations: confirm the office Kafka SASL preference and deployment termination grace period.
- S8: run SQL Server/MySQL outbox integration tests when Docker/test databases are available.

## Close stale questions only with evidence
- The implementation supports PLAIN and both SCRAM variants regardless of the still-unknown office preference.
- JKS and PEM trust material are supported; the JKS parser decision and dependency version are recorded in `14-decisions-assumptions-and-constraints.md` proposal.
- Diagnostics snapshot pause measured 0.229ms upper bound in the recorded 5000-goroutine benchmark, below the 10ms stop limit.
