# go-core — AI Development Principles

`go-core` is an infrastructure foundation library for Go microservices.
Not a business service. Not a generic utils repo. Domain-agnostic. Go 1.24.

## Ownership: What go-core Owns

- Startup composition and runtime wiring
- Config loading and validation (env-driven, 12-factor)
- Lifecycle and graceful shutdown
- Transport wrappers: gRPC + HTTP gateway
- Logging, metrics, and tracing baseline
- Infrastructure connectors: DB, cache, messaging, migration
- Technical error contract and transport mapping
- Selected platform-standard technical contracts intentionally shared across services

## Ownership: What go-core Does NOT Own

- Business entities or domain rules
- Service-specific schema or workflow semantics
- Product-specific naming, aliases, or event payloads
- Hidden automation not controlled by the consuming service
- Generic utilities → those belong in `utils-shared`

## Prompt Roles

| Prompt | Purpose |
|---|---|
| `prompts/breakdown.md` | Plan a task — assess risk and define scope before writing code |
| `prompts/execute.md` | Implement a planned change |
| `prompts/fix.md` | Debug and fix a specific error |
| `prompts/test.md` | Write unit and integration tests |
| `prompts/review.md` | Review code before merge |
| `prompts/new-feature.md` | Add a new module or feature |
| `prompts/security-review.md` | Audit a change touching auth, data, or secrets |
| `prompts/architecture-consult.md` | Get structured analysis on a design decision |

## Execution Principles

- Prefer safe evolution — additive changes first
- Allow bounded refactors that improve the framework shape
- Avoid hidden side effects and undocumented runtime behavior
- Preserve documented exported behavior as the semver contract
- Keep defaults generic — no service-specific names anywhere
- Keep service-specific semantics out of framework code and docs
- Target Go `1.24`
