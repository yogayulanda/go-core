# Domain Boundary

`go-core` has no business domain.

Allowed: technical/runtime domain (health, transport context, infra contracts).
Allowed with explicit intent: platform-standard technical contracts shared by a class of services.
Not allowed: business entities, business rules, service-specific states.

Boundary examples:

- `dbtx` is allowed because it is SQL transaction infrastructure.
- `logger.TransactionLog` is allowed because it is an intentional monitoring contract for transaction-oriented services.
- business workflow steps and business data models still belong in the service repo.

Heuristic: business behavior belongs in the service repo.
