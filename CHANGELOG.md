# Changelog

All notable changes to `go-core` are recorded in this file.

This changelog complements:
- GitHub Release notes for announcement-style summaries
- `MIGRATION.md` for consumer-visible upgrade guidance
- `docs/RELEASING.md` for the release process

## [1.1.0] - Unreleased

### Added
- No public 1.1.0 release changes have been selected. Internal foundation work is tracked below and must not be treated as a published tag.

## [Internal 1.0.0] - Prepared 2026-10-14; not tagged

This section records the internal foundation refresh completed in this workspace. It is not a tag or public release. `D-PATH` remains open; do not consume this state as a published internal module until the path and release gates are approved.

### Added
- Registry-backed response codes, `CORE-1005` for oversized HTTP bodies, and a consistent HTTP success/error envelope.
- Configurable gateway timeouts, CORS allowlists, request identity capture/forwarding, gRPC health, graceful draining, and effective-config redaction.
- HTTP transport errors with before/after-send classification; an opt-in-retry HTTP client; and a new gRPC client with identity propagation, default deadline, metrics, logging, OTel, and retries disabled.
- Kafka PLAIN/SCRAM and JKS/PEM TLS configuration, per-partition ordered consumers, header-preserving DLQ messages, per-key leased outbox delivery, and CloudEvents audit publication.
- Bounded route-template metric labels, optional runtime diagnostic snapshots, and the `coretest` test-helper package.

### Changed (Breaking)
- Database drivers are SQL Server and MySQL only. Removed configuration environment overrides are ignored and reported; runtime tuning is represented by code fields.
- Error-code APIs use an open registry and `errors.New`/`Wrap`/`Validation`; prior closed category/domain/build APIs are not part of this contract.
- Gateway response envelopes and ECS field names are changed. Consumers should read `status`, `response_code`, `message`, `request_id`, `timestamp`, `data`, and validation `details` as applicable.
- Gateway CORS is disabled unless origins are allowlisted; body limits, configured HTTP timeouts, trusted proxy hops, and gRPC/HTTP health semantics are explicit runtime contracts.
- Request identity fields use the context bag and ECS names (`labels.request_id`, `labels.gtid`, `user.id`, `client.ip`, `user_agent.original`); a REST request reaching gRPC has one access line from the gRPC interceptor.
- HTTP client retries are off by default. Received HTTP status responses, including 5xx, return `(response, nil)`; transport failures return `TransportError`.
- HTTP identity propagation is internal-only opt-in; body logging requires a caller masker. The new `grpcclient` is a new outbound contract and defaults to the server's plaintext mode unless transport credentials are supplied.
- The outbox schema is replaced: identity `id`, unique UUID `event_id`, `msg_key`, `status`, `attempt`, `last_error`, `available_at`, `locked_until`, and `sent_at`. Existing outbox tables require a service-owned migration before adopting the worker.
- Kafka consumers serialize processing/commit order within each partition; concurrency remains across partitions.
- Logger and database constructors now use functional options. `logger.NewWithOptions` and `database.NewWithLogConfig` were removed.
- Shutdown now drains before graceful server stop; startup logs the effective config with secret values redacted. `DIAG_ENABLED` adds bounded diagnostic snapshots by default.

The Go module path is unchanged in this prepared state. A consuming internal release still requires `D-PATH` approval and a separate human-controlled release/tag action.

### Known Gaps Before Release
- `GC-23` is resolved as OpenTelemetry OTLP to Elastic APM Server; no APM agent or `transaction.id` field is used.
- `GC-24` is resolved as `METRICS_ENABLED=false` by default; metrics and `/metrics` are opt-in, and `app_transaction_total` is removed.
- `D-PATH` (internal module path) remains open; no tag was created.
- `Q-SIG` confirmation of the existing signature scheme remains deferred.
- The source directive remains Go 1.24.3; the office build image is Go 1.24.13. Go 1.26.x is the next planned toolchain upgrade.
- SQL Server/MySQL outbox integration tests were authored with the `integration` build tag but not run here because Docker was unavailable (S8).

## [1.0.1] - 2026-05-25

### Changed
- Public OSS readiness: professionalized README/onboarding docs, added Apache-2.0 licensing, contributing/security policies, and a safe `.env.example`.
- Removed legacy `.ai` context artifacts from the public tree while retaining `.forge` and `CLAUDE.md` workflow context.
- Sanitized public release-gate examples and fixture naming.

### Compatibility
- No runtime behavior or public Go API changes intended.

## [1.0.0] - 2026-04-10

### Added
- stable `v1.0.0` release baseline for `go-core`
- canonical `.ai` entrypoints:
  - `.ai/context.md`
  - `.ai/architecture.md`
  - `.ai/workflow.md`
- `docs/VERSIONING.md` as the semver and upgrade-discipline reference
- release evidence and release-process alignment for tagged releases

### Changed
- documented `go-core` as a stable foundation runtime for Go services
- aligned README, migration guidance, bootstrap guidance, observability guidance, and production sign-off docs with the stable `v1.0.0` baseline
- verified `/version` metadata expectations through tests and release guidance
- updated GitHub Actions CI to run on branch pushes, tag pushes, and manual dispatch with Node 24-compatible actions

### Notes
- `v1.0.0` is the first public stable compatibility baseline for downstream adopters.
- `v1.0.1` is the public OSS-readiness patch on that line.
- future consumer-visible changes should update both this changelog and `MIGRATION.md` when upgrade behavior changes
