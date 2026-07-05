# Open Questions

## When to Read
- Read whenever a change depends on facts the repository does not currently confirm.

## Do Not Use This For
- Confirmed facts.
- Durable assumptions already accepted in `14-decisions-assumptions-and-constraints.md`.

## Source of Truth
- Real repository gaps that should stop AI guesswork.

## Current Context
- Unknowns belong here instead of being invented in other files.

## Open Questions
- Q-001: Which human/team is the current approval authority for durable context changes and semver-impacting contract changes? The repo docs describe process expectations but do not name owners.
- Q-002: Are there additional current ADRs or decision records outside this repository that govern platform-standard contract additions beyond the documented `TransactionLog` example?
- Q-003: Should `INTERNAL_JWT_ENABLED=true` support JWKS-only runtime configuration without `INTERNAL_JWT_PUBLIC_KEY`, or should docs and context continue to treat the public key as required until validation changes?

## Confirmed Facts
- Consuming services own their business APIs and payload schemas, so the absence of protobuf/OpenAPI files here is intentional rather than a missing contract.

## Assumptions
- Open questions are limited to ownership, external decision-record visibility, and the unresolved JWT validation/config contract gap surfaced by current code/docs.

## Related Files
- `00-index.md`
- `14-decisions-assumptions-and-constraints.md`
- `98-glossary.md`
