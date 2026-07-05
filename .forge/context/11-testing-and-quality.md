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

## Assumptions
- None.

## Related Files
- `10-observability-and-support.md`
- `12-runtime-deployment-and-config.md`
- `13-operations-and-runbook.md`
