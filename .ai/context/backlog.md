Current Prioritized Backlog

## P1

- messaging and outbox runtime behavior
- refresh older examples to match the current golden path

Recently completed:

- runtime orchestration in `app/` and `server/`
- transport alignment in `server/grpc/` and `server/gateway/`

## P2

- cache ergonomics and observability
- error contract hardening
- security observability and diagnosability
- migration adoption workflow

## P3

- resilience observability
- script and quality gate alignment
- versioning and upgrade discipline
- cleanup of legacy `.ai` tasks that no longer reflect the latest repo direction

Backlog rules:

- prefer additive changes first
- keep `.ai/` as the source of truth for `forge`
- improve untouched areas in priority order unless a critical defect overrides it
