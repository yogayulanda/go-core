# go-core Context (Compact)

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

## Does Not Own
- business rules
- service schema/model semantics
- product-specific event payload logic

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
