Common Patterns

Configuration:

- load from env
- normalize values early
- validate all user-facing constraints centrally
- keep aliases and defaults generic across services
- provide structured validation issues without breaking the compact public error path

Runtime:

- initialize dependency
- register lifecycle shutdown
- fail fast only for required dependencies
- expose startup behavior explicitly through app/server wiring
- avoid background automation that is not visible from the consuming service
- keep outbox worker ownership explicit in the consuming service

Testing:

- prefer focused unit tests
- isolate external systems with sqlmock or function overrides
- verify config, runtime, and docs stay aligned
- cover success and failure paths for exported behavior

Documentation patterns:

- examples and templates should reflect the golden path, not just isolated snippets
- docs should describe explicit ownership for optional runtime components
- platform-standard technical contracts should state their target service class explicitly
