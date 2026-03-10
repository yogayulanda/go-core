# Domain Boundary

`go-core` has no business domain.

Allowed: technical/runtime domain (health, transport context, infra contracts).
Not allowed: business entities, business rules, service-specific states.

Heuristic: business vocabulary belongs in service repo.
