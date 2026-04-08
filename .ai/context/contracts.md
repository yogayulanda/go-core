Contract Classes

`go-core` supports two kinds of contracts.

## 1. Generic foundation contracts

These are the default shape of the repository:

- config loading and validation
- runtime composition and lifecycle
- transport wrappers
- tracing, metrics, logging baseline
- database, cache, messaging, migration helpers
- technical error mapping
- SQL transaction orchestration via `dbtx`

## 2. Platform-standard technical contracts

These are allowed when intentionally shared across multiple services in the ecosystem.
They are not generic for every service, but they are still technical contracts rather than business rules.

Approved example:

- `logger.TransactionLog`
- `logger.Logger.LogTransaction(...)`
- `app_transaction_total{service,operation,status}`

Approved generic foundation logging contracts:

- `logger.ServiceLog`
- `logger.Logger.LogService(...)`
- `logger.DBLog`
- `logger.Logger.LogDB(...)`

Approved additive observability contracts:

- `app_service_operation_total{service,operation,status}`
- `app_service_operation_duration_seconds{service,operation}`
- `app_db_operation_total{service,db_name,operation,status}`
- `app_db_operation_duration_seconds{service,db_name,operation}`

Rules:

- keep the surface small and stable
- document target audience clearly
- do not use this as a backdoor for business workflow logic
- do not assume every service must consume the platform-standard contract
- keep `TransactionLog` scoped to transaction-oriented services
- keep additive config DX improvements compatible with `Validate()`
