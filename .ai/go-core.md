# go-core Lens

Use this file as the primary AI lens before reading deeper context.

## Repo Identity
- `go-core` is a reusable infrastructure foundation for Go services.
- It is a foundation framework repo, not a business service repo and not a generic utils dump.
- Keep changes domain-agnostic, explicit, and safe for reuse across multiple services.
- Shared low-level helpers that are merely generic utilities belong in `utils-shared`, not here.
- Selected platform-standard technical contracts may live here when intentionally shared across services.

## Core Responsibility
- bootstrap and lifecycle
- environment config loading and validation
- transport wrappers for gRPC and HTTP gateway
- logging, metrics, tracing, request metadata
- infra connectors and helpers for DB, cache, messaging, migration, retry
- shared technical error contract and mapper
- approved platform observability contracts for specific service classes

## Non-Goals
- business entities or product rules
- service-specific workflow semantics
- product-specific config names, aliases, or event payloads
- hidden automation that downstream services cannot control

## Hard Constraints
- prefer safe evolution; additive changes are preferred, but internal refactors are allowed while adoption is still limited
- keep public API surface small
- avoid hidden background behavior
- preserve explicit lifecycle wiring
- if logic can reasonably live in a consuming service, keep it out of `go-core`

## Contract Classes
- Generic foundation contracts:
  runtime/bootstrap building blocks expected to apply broadly.
- Platform-standard technical contracts:
  intentionally standardized technical contracts for a class of services.

Current approved platform-standard example:
- `logger.TransactionLog`
- `logger.Logger.LogTransaction(...)`
- `app_transaction_total{service,operation,status}`

Current approved generic logging contracts:
- `logger.ServiceLog`
- `logger.Logger.LogService(...)`
- `logger.DBLog`
- `logger.Logger.LogDB(...)`

## Current Stage
- Primary consumer today is `transaction-history-service`.
- Broad backward compatibility is not yet the top constraint.
- Still avoid careless churn in exported behavior unless the foundation becomes cleaner or safer.

## Technical Baseline
- target Go version: `1.24`

## Implementation Bias
- runtime functions should take `ctx context.Context` first
- reuse `errors.AppError` contract when exposing application errors
- sanitize external/API-facing error responses
- keep internal diagnostic detail in logs, not public payloads
- when config/runtime behavior changes, align code, tests, and docs together

## Review Gate
Always check and mention:
- compatibility
- coupling
- concurrency
- scale risk
- overengineering risk
