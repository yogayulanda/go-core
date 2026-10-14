# Proposed Context Patch: GC-28 Go Toolchain Policy

Status: promoted on 2026-10-14 to `.forge/context/12-runtime-deployment-and-config.md` and `.forge/context/14-decisions-assumptions-and-constraints.md`.

## Decision

Replace the Go-version constraints in `.forge/context/14-decisions-assumptions-and-constraints.md` and `.forge/context/12-runtime-deployment-and-config.md` with:

- `GC-28`: Follow a rolling N-1 Go policy. Use the latest patch of the N-1 minor line, advance the target minor with Go's February/August releases, and apply security patches within 14 days.
- The `go` directive in `go.mod` is the minimum required by the selected dependency graph and is determined with `go mod tidy`; it is not automatically the installed Go patch version.
- Prefer stable dependency releases older than 14 days. An official maintainer's minimum security fix may be younger when needed to remediate an advisory; record the release date, age, and exception.

## Evidence

- `go.mod` declares `go 1.26.0`, the minimum required by selected dependencies after `GOTOOLCHAIN=go1.26.9 go mod tidy`.
- CI pins Go 1.26.9 for toolchain validation.
- `docs/SECURITY_ADVISORIES.md` records the 2026-10-14 scan and the approved six-day-old x/net v0.60.0 security fix exception.
- `MIGRATION.md` documents consumer requirements and behavior checks.

## Supersedes

The existing active-context constraint to retain `go 1.24.3`, use Go 1.24.13 as the image, and target Go 1.26.x next, plus the runtime context statement that the module targets Go 1.24.3.

Do not promote this proposal without maintainer review.