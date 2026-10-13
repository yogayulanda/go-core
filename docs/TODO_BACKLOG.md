# TODO Backlog

This backlog tracks work intentionally left outside the completed internal foundation batches. Do not interpret deferred items as implemented defaults.

## Release-Blocking Decisions

- `GC-23`: choose OTLP-to-Elastic APM versus the Elastic APM Go agent; current code uses the existing OTLP gRPC exporter configured by `APM_ENDPOINT`.
- `GC-24`: decide metrics enablement policy/default and whether `app_transaction_total` is removed or incremented. The metric is currently registered but not incremented.
- `D-PATH`: approve the internal module path and perform the human-controlled path/tag release work (`WS-9`). No tag or module-path change was made in this workspace.
- `Q-SIG`: confirm the existing HTTP signature scheme with a consuming service before changing its wire contract.

## Operational Follow-Up

- Run `go test -tags=integration ./messaging/outbox/...` against disposable SQL Server and MySQL instances. Integration tests compile but were not run in the implementation environment because Docker was unavailable (S8).
- Confirm which SASL mechanism the office Kafka cluster uses. The library supports PLAIN, SCRAM-SHA-256, and SCRAM-SHA-512.
- Review and remediate the current govulncheck advisories in pre-existing dependency versions; WS-6 did not upgrade unrelated modules.
- Confirm deployment termination grace period exceeds `DrainDelay + ShutdownTimeout` (default total 28 seconds, below the common 30-second value but leaves little margin).

## Deferred Platform Work

- GitLab CI (`GC-02`) is not part of the approved internal release scope; current CI remains GitHub Actions.
- Inbox/idempotency helper (`GC-37`) is explicitly excluded.
- Dashboard, alert thresholds, data retention, and client-IP privacy retention remain deployment/platform responsibilities.
