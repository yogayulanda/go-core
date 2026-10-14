# TODO Backlog

This backlog tracks work intentionally left outside the completed foundation updates. Do not interpret deferred items as implemented defaults.

## Deferred Decisions

- Downstream forks may choose an import module path before adoption; this does not block the public release.
- Q-SIG confirmation is deferred and is not a release blocker; do not change the current signature contract without compatibility evidence.

## Operational Follow-Up

- Run `go test -tags=integration ./messaging/outbox/...` against disposable SQL Server and MySQL instances. Integration tests compile but were not run in the implementation environment because Docker was unavailable (S8).
- Confirm which SASL mechanism the deployment's Kafka cluster uses. The library supports PLAIN, SCRAM-SHA-256, and SCRAM-SHA-512.
- Reassess residual dependency advisories in `SECURITY_ADVISORIES.md` when a Go 1.26.x build image is available; current fixes requiring Go 1.25+ are intentionally deferred.
- Confirm deployment termination grace period exceeds `DrainDelay + ShutdownTimeout` (default total 28 seconds, below the common 30-second value but leaves little margin).

## Deferred Platform Work

- Go 1.26.x toolchain upgrade requires a new build image. Keep `go 1.24.3` in `go.mod`; the current image target is Go 1.24.13.
- GitLab CI (`GC-02`): add the separately approved GitLab pipeline; current CI remains GitHub Actions.
- Migrate golangci-lint to v2 after compatibility and config review; current validated version is v1.64.8.
- Inbox helper (`GC-37`) and protovalidate integration (`GC-38`) remain deferred.
- `GC-23` and `GC-24` are resolved: OTLP to Elastic APM Server without an agent; metrics default false, endpoint/recording gated, transaction counter removed.
- Dashboard, alert thresholds, data retention, and client-IP privacy retention remain deployment/platform responsibilities.
