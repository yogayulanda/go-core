# go-core Context

## Repository Purpose
`go-core` is the stable infrastructure foundation for Go services in this ecosystem.

It owns reusable runtime concerns:
- bootstrap and lifecycle
- environment-driven config loading and validation
- gRPC and HTTP gateway wrappers
- logging, metrics, tracing, request metadata
- database, cache, messaging, migration, and resilience helpers
- shared technical error contract
- selected platform-standard technical contracts when intentionally standardized

It does not own:
- business entities or product rules
- service-specific workflow semantics
- product-specific payload or schema design
- generic helper utilities that belong in `utils-shared`

## Stabilization State
This repository is in the stable `v1.0.0` state.

Meaning:
- exported behavior should now be treated as a stable foundation contract
- semver applies to public API, config behavior, runtime behavior, transport-facing behavior, and documented observability contracts
- changes should prefer additive evolution
- breaking changes require major-version intent and explicit migration notes

## Public Contract Classes
1. Generic foundation contracts
   Reusable building blocks expected to apply broadly across services.

2. Platform-standard technical contracts
   Technical contracts intentionally standardized for a class of services.

Current approved platform-standard example:
- `logger.TransactionLog`
- `logger.Logger.LogTransaction(...)`
- `app_transaction_total{service,operation,status}`

## Development Defaults
- keep changes minimal and bounded
- preserve explicit lifecycle ownership
- avoid hidden runtime automation
- keep optional dependencies opt-in and service-owned
- align code, docs, tests, and migration notes when public behavior changes

## Primary References
- `README.md`
- `docs/FOUNDATION_BOUNDARY.md`
- `docs/ARCHITECTURE.md`
- `docs/SERVICE_BOOTSTRAP.md`
- `docs/OBSERVABILITY.md`
- `docs/VERSIONING.md`
- `MIGRATION.md`
