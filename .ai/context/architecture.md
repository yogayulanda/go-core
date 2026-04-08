Architecture: framework library

This repository is the runtime foundation for service applications.
It should provide composition, startup, lifecycle, transport, and infra contracts.
It should not become a bucket for unrelated shared helpers.
It may contain selected platform-standard technical contracts when they are intentionally shared.

Core layering:

- app container and lifecycle
- config and validation
- transport wrappers
- observability and logging baseline
- infra connectors and operational helpers
- shared operational contracts
- examples and templates for downstream usage

Rules:

- keep modules loosely coupled
- avoid service-specific assumptions
- avoid hidden background behavior
- keep public interfaces small and explicit
- prefer composition over implicit magic
- prefer framework primitives over catch-all helpers
