# go-core · AI Context

> **Primary lens for all AI-assisted work. Read this first. Always.**

## Project Overview

`go-core` is the **production-grade infrastructure foundation** for all Go microservices in this ecosystem.
It is a framework/library repo — not a business service, not a generic utils dump.

---

## Core Architecture

```
config → app.New() → server.Run()
           |
    ┌──────┼──────────────────────┐
  logger  metrics  lifecycle  databases
           |
    ┌──────┼───────────────────┐
  gRPC   gateway   security   outbox
```

**Bootstrap order (golden path):**
1. `config.Load(...)` — parse env → validate
2. `app.New(ctx, cfg)` — wire logger, metrics, DB, cache, lifecycle
3. Register gRPC/HTTP handlers
4. `server.Run(...)` — start transports, block, graceful shutdown

---

## Key Modules

| Module | Responsibility |
|---|---|
| `app/` | Container, lifecycle, dependency wiring |
| `config/` | Env loading, typed struct, strict validation |
| `server/grpc/` | gRPC transport, auth interceptors, request metrics |
| `server/gateway/` | HTTP gateway, signature validation, panic recovery |
| `security/` | JWT verification (RS256/384/512 + JWKS), claims extraction |
| `errors/` | `AppError` canonical error contract + gRPC mapper |
| `logger/` | Zap structured logging: `ServiceLog`, `DBLog`, `TransactionLog` |
| `observability/` | Prometheus metrics, OTEL tracing |
| `dbtx/` | SQL transaction manager + context propagation |
| `messaging/` | Kafka publisher, consumer, outbox worker |
| `resilience/` | Retry (exp backoff + jitter), circuit breaker, timeout |
| `httpclient/` | Resilient outbound HTTP client (resty + CB + retry + OTEL) |
| `cache/` | Redis, Memcached adapters |
| `migration/` | Goose migration autorun with distributed lock |

---

## Entry Points

- `app.New(ctx, cfg)` — application bootstrap
- `server.Run(...)` — transport orchestration
- `config.Load(...)` — configuration initialization
- `errors.AppError` — canonical error type

---

## Critical Rules

### Security
- **Never** expose internal error details in API responses — sanitize at transport boundary
- JWT verification requires RS algorithm; symmetric (HS*) is blocked by `ValidMethods` constraint
- Sensitive fields are auto-redacted in logs (`password`, `token`, `card`, `otp`, `cvv`, `pin`, `secret`)
- Metadata-only mode (`INTERNAL_JWT_ENABLED=false`) is **non-enforcing** — intended for trusted internal calls only

### Transactions
- Boundary: `dbtx.WithTx(ctx, db, fn)` — wraps commit/rollback/panic recovery
- Repository must use `dbtx.FromContext(ctx)` to reuse the active transaction
- Outbox record **must** be written in the **same transaction** as domain data
- Keep transactions short; no network calls inside a DB transaction

### Stability
- This repo is at stable **v1.0.0** — all public API changes require semver intent
- Prefer additive changes; breaking changes require major-version bump + `MIGRATION.md` entry
- Keep `go-core` domain-agnostic — no business entities, no service-specific defaults

---

## What Does NOT Belong Here

- Business entities, product domain rules
- Service-specific workflow logic or event semantics
- Generic utility helpers → use `utils-shared`
- Hidden background automation not controlled by the consuming service

---

## Primary References

| File | Purpose |
|---|---|
| `.ai/architecture.md` | System design, layers, scaling |
| `.ai/security.md` | Auth flow, JWT, secrets handling |
| `.ai/transactions.md` | Transaction flow, idempotency, failure handling |
| `.ai/modules.md` | Module details, APIs, conventions |
| `.ai/data-flow.md` | Request lifecycle, observability pipeline |
| `.ai/integrations.md` | External services, dependencies |
| `.ai/conventions.md` | Code style, patterns, engineering rules |
| `.ai/decisions.md` | Architecture decision records |
| `docs/` | Framework guidance documents |
| `MIGRATION.md` | Upgrade notes |
