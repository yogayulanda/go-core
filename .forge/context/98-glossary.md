# Glossary

## When to Read
- Read when repo-local technical terms or shared contracts are unclear.

## Do Not Use This For
- Open unknowns: `99-open-questions.md`.
- Full behavioral explanations already owned by other context files.

## Source of Truth
- Stable repository-specific terms and acronyms used across the foundation.

## Current Context
- Scope: durable technical terms used by `go-core` docs and code.

## Confirmed Facts
- `go-core`: the reusable Go backend foundation module in this repository.
- `AppError`: the canonical application-visible error type defined in `errors/`.
- `ServiceLog`: the shared structured runtime log contract used for service-flow and orchestration events.
- `DBLog`: the structured database operational/query log contract.
- `TransactionLog`: the approved platform-standard log contract for transaction-oriented services only.
- `Outbox`: the SQL-backed deferred event publication pattern implemented under `messaging/outbox/`.
- `JWKS`: JSON Web Key Set endpoint used as an alternative to a static RSA public key for JWT verification.

## Assumptions
- None.

## Related Files
- `04-interfaces-and-contracts.md`
- `08-security-and-access.md`
- `14-decisions-assumptions-and-constraints.md`
