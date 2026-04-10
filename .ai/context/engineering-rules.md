Engineering Rules

Use generic naming in config, docs, and examples.

Do not hardcode service-specific database aliases such as `transaction`.

Any new default must be framework-safe across multiple services.

Public API changes require tests and README updates.
Public contract changes that affect upgrades must update `MIGRATION.md`.

Runtime APIs should take `ctx context.Context` first when applicable.

Prefer reusing `errors.AppError` instead of inventing parallel app-error contracts.

Target Go version is `1.24`.

When changing config behavior:

- update loader
- update validation
- update docs
- update tests

When changing runtime wiring:

- avoid hidden side effects
- preserve explicit lifecycle registration

When changing release-facing behavior:

- align CI, release docs, and scripts
- keep version metadata and upgrade guidance explicit
