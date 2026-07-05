# Domain Boundaries

## When to Read
- Read before adding new shared contracts, platform abstractions, or service-facing runtime behavior.

## Do Not Use This For
- Foundation flow rules: `06-business-rules-and-flows.md`.
- Technical model ownership: `05-data-and-persistence.md`.

## Source of Truth
- What `go-core` is allowed to own versus what must stay in consuming service repositories.

## Current Context
- `go-core` explicitly has no business domain.
- The repo may own technical/runtime contracts such as config loading, transport wrappers, transaction helpers, technical logging, and observability baselines.

## Confirmed Facts
- Business entities, business rules, service-specific workflow branching, and product-specific payload contracts are out of scope and belong in consuming services.
- Platform-standard technical contracts are allowed only with explicit intent; the current documented example is `logger.TransactionLog` and its matching `app_transaction_total{service,operation,status}` metric family.
- SQL transaction orchestration via `dbtx` is in scope because it is infrastructure behavior, not business logic.

## Assumptions
- None.

## Related Files
- `01-service-overview.md`
- `06-business-rules-and-flows.md`
- `14-decisions-assumptions-and-constraints.md`
