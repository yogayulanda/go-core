# Security and Access

## When to Read
- Read before changing auth mode, secret-bearing config, signature validation, or client-visible auth failures.

## Do Not Use This For
- General dependency inventory: `07-integrations-and-dependencies.md`.
- Runtime recovery behavior: `09-errors-and-resilience.md`.

## Source of Truth
- Technical auth modes, metadata/JWT claim handling, and secret-safety rules in the foundation.

## Current Context
- `go-core` supports non-enforcing metadata extraction mode and enforcing internal JWT verification mode for gRPC requests, but the JWT config path still has an unresolved JWKS-only validation ambiguity.

## Confirmed Facts
- Metadata extraction mode reads `x-subject`, `x-session-id`, `x-role`, and `x-claim-<name>` keys into `security.Claims`.
- JWT verification mode validates RSA-signed tokens, can enforce optional issuer and audience checks, and prefers a configured JWKS endpoint before falling back to RSA public key parsing in `security.NewInternalJWTVerifier`.
- Config validation still requires `INTERNAL_JWT_PUBLIC_KEY` whenever `INTERNAL_JWT_ENABLED=true`, so a JWKS-only configuration is not a confirmed fully supported validated runtime option yet.
- JWT auth policy supports include-list or exclude-list method matching through `INTERNAL_JWT_INCLUDE_METHODS` / `INTERNAL_JWT_EXCLUDE_METHODS` config fields represented in `config.InternalJWTConfig`.
- Gateway middleware can enforce request-signature validation when `AUTH_SIGNATURE_ENABLED` is set.
- Docs require that secrets and raw tokens never be exposed in logs or returned errors.
- gRPC startup logs auth configuration mode for operator visibility, while client-facing failures stay sanitized as unauthorized responses.

## Assumptions
- Authorization roles and service-specific permission semantics are consumer-service responsibilities, not foundation-owned rules.

## Related Files
- `04-interfaces-and-contracts.md`
- `09-errors-and-resilience.md`
- `12-runtime-deployment-and-config.md`
