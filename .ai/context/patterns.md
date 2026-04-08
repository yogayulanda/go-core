Common Patterns

Configuration:

- load from env
- normalize values early
- validate all user-facing constraints centrally
- keep aliases and defaults generic across services

Runtime:

- initialize dependency
- register lifecycle shutdown
- fail fast only for required dependencies
- expose startup behavior explicitly through app/server wiring
- avoid background automation that is not visible from the consuming service

Testing:

- prefer focused unit tests
- isolate external systems with sqlmock or function overrides
- verify config, runtime, and docs stay aligned
- cover success and failure paths for exported behavior

Boundary:

- business behavior belongs in consuming services
- generic helper code belongs in `utils-shared`
- `go-core` should keep framework-level composition and technical contracts
