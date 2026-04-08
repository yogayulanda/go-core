Engineering Rules

Always keep `go-core` domain-agnostic.

Prefer additive changes over breaking changes.
While adoption is still limited, coherent refactors are acceptable if they improve the foundation shape.

Use generic naming in config, docs, and examples.

Do not hardcode service-specific database aliases such as `transaction`.

Any new default must be framework-safe across multiple services.

Keep generic helper code out of this repo when it belongs in `utils-shared`.

Public API changes require tests and README updates.

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

When changing foundation boundaries:

- ask whether the change belongs in a service instead
- ask whether the change is actually a generic utility better suited for `utils-shared`
