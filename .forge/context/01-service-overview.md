# Service Overview

## When to Read
- Read before changing repository purpose, supported foundation capabilities, or adoption guidance.

## Do Not Use This For
- Detailed package boundaries: `02-architecture.md`.
- Transport and event contract details: `04-interfaces-and-contracts.md`.

## Source of Truth
- Repository purpose, supported runtime capabilities, intended adopters, and explicit non-goals.

## Current Context
- `go-core` is a reusable Go backend foundation module for production-oriented services.
- The module path is `github.com/yogayulanda/go-core`.
- The repo provides runtime building blocks for config, app lifecycle, transport wiring, logging, metrics, tracing, database access, messaging, migrations, cache, resilience, and error handling.

## Confirmed Facts
- The repo is infrastructure-focused and intentionally does not own service business logic, persistence models, API contracts, deployment topology, or operational policy.
- The documented bootstrap path is: `config.Load(...)` -> `cfg.Validate()` -> optional `migration.AutoRunUp(...)` -> `app.New(...)` -> transport construction -> `server.Run(...)`.
- Examples under `examples/` are starter references for bootstrap, repositories, services, HTTP clients, logging, pagination, and outbox usage.

## Assumptions
- None.

## Related Files
- `02-architecture.md`
- `03-domain-boundaries.md`
- `12-runtime-deployment-and-config.md`
