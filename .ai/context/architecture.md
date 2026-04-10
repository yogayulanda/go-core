Architecture: framework library

Core layering:

- app container and lifecycle
- config and validation
- transport wrappers
- observability and logging baseline
- infra connectors and operational helpers
- shared operational contracts
- examples and templates for downstream usage

High-risk architectural areas:

- `app/`
- `config/`
- `migration/`
- `server/`

Boundary-sensitive areas:

- `errors/`
- `security/`
- `logger/`
- `observability/`
