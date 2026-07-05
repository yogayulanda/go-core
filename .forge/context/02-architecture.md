# Architecture

## When to Read
- Read before refactors, module moves, or changes to runtime composition.

## Do Not Use This For
- Business-domain ownership rules: `03-domain-boundaries.md`.
- Environment and release configuration detail: `12-runtime-deployment-and-config.md`.

## Source of Truth
- Architecture shape, major packages, runtime path, and extension rules.

## Current Context
- The repo is a Go module that exposes foundation packages rather than a standalone business service.
- Runtime composition is documented as config load/validation, optional migration autorun, `app.App` construction, transport setup, then `server.Run(...)` orchestration with graceful shutdown.

## Confirmed Facts
- `app/` builds the runtime container, optional dependency clients, lifecycle hooks, and helper constructors for Kafka publisher/consumer.
- `server/grpc/` and `server/gateway/` wrap transport setup, middleware/interceptors, startup logging, and graceful shutdown integration.
- `config/` owns environment-driven configuration loading and validation; `database/`, `cache/`, `messaging/`, `migration/`, `resilience/`, `httpclient/`, `logger/`, `observability/`, and `security/` provide focused technical subsystems.
- `server.Run(...)` starts components concurrently, cancels on component failure or caller cancellation, and waits for graceful shutdown with a bounded timeout.
- `docs/FOUNDATION_BOUNDARY.md` and `docs/ARCHITECTURE.md` define the extension rule: add capabilities only when they are reusable foundation behavior or an approved platform-standard technical contract.

## Assumptions
- None.

## Related Files
- `03-domain-boundaries.md`
- `07-integrations-and-dependencies.md`
- `14-decisions-assumptions-and-constraints.md`
