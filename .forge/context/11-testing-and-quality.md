# Testing and Quality

## When to Read
- Read before changing validation depth, release gates, or regression expectations.

## Do Not Use This For
- Runtime configuration details: `12-runtime-deployment-and-config.md`.
- Business behavior ownership: `06-business-rules-and-flows.md`.

## Source of Truth
- Test surfaces, CI baseline, and stronger local/release gates for this repository.

## Current Context
- The repo has broad package-level test coverage, a GitHub Actions CI workflow, and release-oriented shell gates under `scripts/`.

## Confirmed Facts
- `make test`, `make vet`, and `make lint` are the standard fast local commands; `make check` runs all three.
- `.github/workflows/ci.yml` runs `make test`, `make vet`, and `make lint` on push, pull request, and manual dispatch.
- `make quality-gate` is the stronger release-oriented gate and the docs require it to include `go test ./...`, `go test -race ./...`, `go vet ./...`, `golangci-lint run`, and `golangci-lint run -E gosec --tests=false`.
- Release docs also define smoke, performance, and failure-drill gates plus a release evidence template.
- Tests exist across core packages including app lifecycle, config validation, database, dbtx, errors, http client, logger, messaging, migration, observability, resilience, security, server, and version.
- `coretest` provides recording logger, fake publisher, response-registry cleanup, and reusable HTTP behavior fixtures; its examples are exercised by tests.
- SQL Server/MySQL outbox integration tests use the `integration` build tag and require disposable real databases. They compile, but real-database verification remains open because Docker and DSNs were unavailable in the implementation environment.
- Diagnostics include periodic-check and 5000-goroutine snapshot benchmarks. The snapshot benchmark fails when measured GC-pause upper bound exceeds 10 ms; runtime snapshots do not use this as a guard.
- Current vulnerability findings and Go-toolchain requirements are documented in `docs/SECURITY_ADVISORIES.md`.

## Assumptions
- None.

## Related Files
- `10-observability-and-support.md`
- `12-runtime-deployment-and-config.md`
- `13-operations-and-runbook.md`
