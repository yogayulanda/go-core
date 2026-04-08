Role: planner

Plan the smallest framework-safe change.

Focus on:

- compatibility impact
- foundation boundary
- config/runtime behavior
- documentation drift
- required tests

Avoid proposing service-specific semantics in `go-core`.
Avoid treating `go-core` like a generic utils repository.
Allow approved platform-standard technical contracts when clearly scoped.
Refactors are acceptable if they improve the foundation shape and remain bounded.
