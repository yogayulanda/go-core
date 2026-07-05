# Data and Persistence

## When to Read
- Read before changing SQL startup behavior, migration helpers, transaction handling, or outbox persistence.

## Do Not Use This For
- Service-owned business data rules: `06-business-rules-and-flows.md`.
- Runtime env details: `12-runtime-deployment-and-config.md`.

## Source of Truth
- Technical models, SQL helper behavior, migration policy, and transaction/outbox persistence rules owned by the foundation.

## Current Context
- `go-core` owns technical persistence helpers, not business tables or domain entities.

## Confirmed Facts
- The documented technical models are `config.Config`, `errors.AppError` / `errors.ErrorResponse`, `messaging.Message`, and outbox `Event`.
- Database startup supports multiple named databases from `DB_LIST`; `app.New(...)` initializes each configured database and fails fast only for databases marked `Required`.
- Migration autorun uses Goose and can acquire a database-specific lock before `up` migrations; the implementation has dedicated lock paths for SQL Server, MySQL, and Postgres.
- SQL Server startup ensures a `dbo.goose_db_version` table exists before Goose runs.
- Transaction guidance is explicit: start the transaction at the use-case boundary with `dbtx.WithTx(...)`, reuse it in repositories with `dbtx.FromContext(ctx)`, keep transactions short, and write outbox records in the same SQL transaction when using the outbox pattern.
- Outbox persistence lives under `messaging/outbox/`; worker startup is intentionally service-owned rather than automatic.

## Assumptions
- None.

## Related Files
- `06-business-rules-and-flows.md`
- `07-integrations-and-dependencies.md`
- `12-runtime-deployment-and-config.md`
