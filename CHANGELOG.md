# Changelog

All notable changes to `go-core` are recorded in this file.

This changelog complements:
- GitHub Release notes for announcement-style summaries
- `MIGRATION.md` for consumer-visible upgrade guidance
- `docs/RELEASING.md` for the release process

## [1.1.0] - Unreleased

### Added
- Resilient `httpclient` package with retries and circuit breaker.
- GORM support in `database` package for SQL Server.
- SASL Plain and JKS certificate support in Kafka configuration.
- Legacy `MEMCACHE_HOST` and `MEMCACHE_PORT` fallback support in Memcached configuration.

### Changed
- Improved `database.New` to initialize a GORM instance by default.

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
- `v1.0.0` is the first stable compatibility baseline for downstream adopters
- future consumer-visible changes should update both this changelog and `MIGRATION.md` when upgrade behavior changes

### Added
- Error classification enums and formatting in `errors` package (`Finality`, `Category`, `Domain`).
- Standardized logging keys in `logger` package (`FieldCustomerID`, `FieldIdempotencyKey`, etc.) and zap injection for `transaction_id`.

### Changed
- `errors.AppError` and `ErrorResponse` updated to the new standard 9-field shape.
- gRPC mapper now automatically packs and unpacks extended metadata.
- HTTP Gateway error handler grabs `trace_id` and `transaction_id` from observability context.
