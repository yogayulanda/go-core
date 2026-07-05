# Business Rules and Flows

## When to Read
- Read before changing foundation control flow that consuming services depend on.

## Do Not Use This For
- Service business behavior; that belongs outside this repository.
- Wire-format definitions: `04-interfaces-and-contracts.md`.

## Source of Truth
- Technical flow rules and behavioral guidance that `go-core` treats as stable foundation behavior.

## Current Context
- The repo has no business workflow rules; its durable rules are technical runtime and integration patterns.

## Confirmed Facts
- Golden-path startup is config load, config validation, optional migration autorun, app construction, transport registration, and `server.Run(...)`.
- `server.Run(...)` starts all provided components plus the app runtime concurrently, cancels the shared context when a component fails, and reports final orchestration status through structured logs.
- Optional infrastructure remains explicit and service-controlled: Redis, Memcached, Kafka, startup migration, and outbox worker execution are opt-in rather than hidden background behavior.
- Direct Kafka publishing is recommended only when the SQL write and event do not need atomic durability; otherwise services should write domain data and an outbox record in the same transaction and let an outbox worker publish later.
- Transaction rules require `dbtx.WithTx(...)` at the use-case boundary and discourage network calls inside SQL transactions.

## Assumptions
- None.

## Related Files
- `03-domain-boundaries.md`
- `05-data-and-persistence.md`
- `09-errors-and-resilience.md`
