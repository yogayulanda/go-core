# Decisions Assumptions and Constraints

## When to Read
- Read before broadening repository scope, adding new shared contracts, or changing release-sensitive behavior.

## Do Not Use This For
- Open unknowns: `99-open-questions.md`.
- Stable term definitions: `98-glossary.md`.

## Source of Truth
- Cross-cutting repository decisions, durable assumptions, and constraints that shape future implementation work.

## Current Context
- This file holds cross-cutting rules that apply across packages and docs.

## Confirmed Facts
- `go-core` is constrained to reusable foundation/runtime behavior and explicitly excludes business entities, business rules, and service-specific workflow logic.
- Optional infrastructure must remain explicit and service-controlled; the foundation should not hide background behavior such as outbox worker startup or optional dependency activation.
- Public-contract changes must update README/docs/tests and `MIGRATION.md` when upgrade behavior changes.
- Release discipline assumes semantic versioning from `v1.0.0` onward and treats tagged releases as the adoption source of truth.
- Current auth config validation requires `INTERNAL_JWT_PUBLIC_KEY` whenever `INTERNAL_JWT_ENABLED=true`, even when `INTERNAL_JWT_JWKS_ENDPOINT` is also set.

## Assumptions
- Consuming services own deployment topology, secret management, authorization policy, and business payload compatibility.

## Related Files
- `03-domain-boundaries.md`
- `11-testing-and-quality.md`
- `99-open-questions.md`
