# Foundation Boundary

`go-core` is the standard runtime foundation for Go services in this ecosystem.

## Allowed Contract Classes

### 1. Generic foundation contracts

These are reusable building blocks expected to make sense for most services:

- config loading and validation
- app lifecycle and shutdown
- gRPC and HTTP gateway wrappers
- logging, tracing, metrics baseline
- database, cache, messaging, migration helpers
- technical error contract and mapping
- SQL transaction orchestration via `dbtx`

### 2. Platform-standard technical contracts

These are not generic for every service, but are intentionally standardized across a class of services.

Current approved example:

- `logger.TransactionLog`
- `logger.Logger.LogTransaction(...)`
- `app_transaction_total{service,operation,status}`

Approved generic logging contracts:

- `logger.ServiceLog`
- `logger.DBLog`

This transaction observability contract exists for transaction-oriented services.
It is a technical monitoring standard, not a business workflow engine.

## Not Allowed

- business entities and business rules
- service-specific workflow branching
- product-specific payload contracts
- generic helper utilities that should live in `utils-shared`

## Decision Gate

Ask these questions before adding something to `go-core`:

1. Is this runtime/foundation behavior that multiple services will consume directly?
2. If not generic for all services, is it still an intentional platform-standard technical contract?
3. Would this be better implemented in a consuming service instead?
4. Is this really just a reusable helper that belongs in `utils-shared`?
