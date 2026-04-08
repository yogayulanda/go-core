# go-core Context (Compact)

Primary source of truth now lives in `.ai/`.
Use `.ai/go-core.md` first, then `.ai/context/*` for current repository guidance.

## Purpose
`go-core` = reusable infra foundation for Go services.
Keep it domain-agnostic and stable.

## Owns
- lifecycle + graceful shutdown
- env config + validation
- logger/metrics/tracing baseline
- transport wrappers (`grpc`, `gateway`)
- infra contracts (db/cache/messaging)
- error contract and mappers
- selected platform-standard technical contracts when intentionally shared across services

## Does Not Own
- business rules
- service schema/model semantics
- product-specific event payload logic
- generic helper utilities better placed in `utils-shared`

## Contract Classes
- Generic foundation contracts:
  config, lifecycle, transport, infra connectors, error mapping, `dbtx`
- Platform-standard technical contracts:
  observability/logging standards that are intentionally shared, including `logger.TransactionLog` for transaction-oriented services

## Current Core Modules
`app`, `config`, `server`, `server/grpc`, `server/gateway`, `errors`, `database`, `dbtx`, `migration`, `cache`, `messaging`, `messaging/outbox`, `observability`, `security`, `resilience`.

## Service Alignment (transaction-history-service)
Consume `go-core` from service layers:
- `internal/domain`: entities + repo contract + domain errors
- `internal/service`: use-case orchestration
- `internal/repository`: SQL/GORM + `dbtx.FromContext`
- `internal/handler/grpc`: request validation + `errors.ToGRPC`

## Decision Gate
If feature can live in service, keep it out of `go-core`.
