# go-core · Architecture Decisions

Architecture Decision Records (ADRs) for `go-core`.
Each decision documents the context, choice, and rationale.

---

## ADR-001: gRPC-Gateway over Pure HTTP Framework

**Status:** Accepted (v1.0.0)

**Context:**
Services need both gRPC (internal) and REST/JSON (external) transport. Options considered: Gin, Fiber, Echo + manual gRPC, or grpc-gateway.

**Decision:**
Use `grpc-ecosystem/grpc-gateway/v2`. HTTP is a projection of the gRPC protobuf contract.

**Rationale:**
- Single protobuf schema serves as source of truth for both transports
- Auth, metrics, and tracing interceptors apply once at gRPC level
- Eliminates duplicate validation and serialization logic
- Strong typing enforcement across transport boundary

**Trade-off:**
- Imposes protobuf requirement on all services
- Less flexible than Gin/Fiber for purely HTTP-native services
- Not suitable for services that genuinely don't need gRPC

---

## ADR-002: Environment Variables Only (12-Factor Config)

**Status:** Accepted (v1.0.0)

**Context:**
Config could be loaded from files (YAML/JSON), env vars, or a config server (Vault, AWS SSM).

**Decision:**
All runtime configuration via environment variables only. Viper used for env parsing. No structured file configs for runtime.

**Rationale:**
- 12-factor compliance — identical binary works across dev/staging/prod
- Simpler K8s ConfigMap/Secret injection
- No config drift between environments
- No file access requirements at runtime

**Trade-off:**
- Large configs result in many env vars
- Nested config (DB per alias) requires naming convention discipline

---

## ADR-003: Context-Based Transaction Propagation (`dbtx`)

**Status:** Accepted (v1.0.0)

**Context:**
SQL transactions need to flow from use-case boundary down through repository layers. Options: explicit `*sql.Tx` parameter, or context injection.

**Decision:**
`dbtx.Inject(ctx, tx)` + `dbtx.FromContext(ctx)` pattern. Repositories receive context, extract `*sql.Tx` if present.

**Rationale:**
- Eliminates `*sql.Tx` pollution through function signatures
- Repositories don't change signature when called inside vs. outside a transaction
- Fits idiomatic Go context propagation patterns
- `dbtx.WithTx` provides automatic commit/rollback/panic recovery

**Trade-off:**
- Implicit dependency — developers must know to use `dbtx.FromContext`
- Requires discipline: repositories must always use context-extracted connection

---

## ADR-004: Explicit Lifecycle Ownership (No Hidden Goroutines)

**Status:** Accepted (v1.0.0)

**Context:**
Infra packages could auto-start background workers (e.g., outbox worker, Kafka consumer) during initialization.

**Decision:**
`go-core` never starts background goroutines automatically. All workers and lifecycle hooks must be explicitly registered by the consuming service.

**Rationale:**
- Services must control which processes run in which pods (workers vs. handlers)
- Hidden automation creates non-obvious resource consumption
- Explicit ownership makes shutdown behavior predictable
- Outbox worker is a prime example: not all service replicas should poll

**Trade-off:**
- More boilerplate in service `main.go`
- Requires documentation and templates to guide correct usage

---

## ADR-005: Centralized Error Contract (`errors.AppError`)

**Status:** Accepted (v1.0.0)

**Context:**
Multiple services need consistent error responses. Without a canonical contract, each handler invents its own error format.

**Decision:**
`errors.AppError{Code, Message, Category, Details, Err}` is the single error type for all application errors. `Err` (internal) is never exposed to clients.

**Rationale:**
- Protects API consumers from leaking SQL internals, stack traces, or sensitive paths
- Consistent error codes enable frontend/mobile error handling
- `ToGRPC()` mapping ensures transport-level correctness
- Category enables structured monitoring without exposing implementation details

**Trade-off:**
- Forces all services to learn and use `AppError`
- New error codes require repo-level changes

---

## ADR-006: `TransactionLog` as Platform-Standard Contract

**Status:** Accepted (v1.0.0)

**Context:**
Transaction-oriented services (payments, transfers) need consistent observability for business-level flows — separate from technical service logs.

**Decision:**
`logger.TransactionLog` and `logger.Logger.LogTransaction(...)` are approved platform-standard contracts in `go-core`, not generic utilities.

**Rationale:**
- Enables unified Grafana dashboards across all transaction services
- Stable `app_transaction_total{service,operation,status}` metric for cross-service alerting
- `UserID`, `TransactionID`, `ErrorCode` are cross-service correlation fields
- Clearly scoped to transaction-oriented services — not imposed on all services

**Trade-off:**
- Breaks the strict "no product-specific code in go-core" rule (intentional exception)
- New services must understand this is optional, not mandatory

---

## ADR-007: Outbox Pattern for Durable Event Delivery

**Status:** Accepted (v1.0.0)

**Context:**
Services that write to a DB and publish a Kafka event risk losing the event if the process crashes between commit and publish.

**Decision:**
`messaging/outbox` provides transactional outbox. Domain data + outbox record written in one DB transaction. Worker publishes separately.

**Rationale:**
- Eliminates the dual-write problem (DB commit + Kafka publish in sequence)
- At-least-once delivery with explicit retry control
- Worker ownership stays with the service — `go-core` only provides the mechanism

**Trade-off:**
- Adds operational complexity (outbox table, worker process)
- At-least-once requires consumer-side idempotency
- Worker must run somewhere — services must plan pod topology

---

## ADR-008: Asymmetric JWT Only (RS256/384/512)

**Status:** Accepted (v1.0.0)

**Context:**
JWT can use symmetric (HMAC: HS256) or asymmetric (RSA: RS256) signing. Symmetric is simpler but requires shared secrets.

**Decision:**
Only RS256, RS384, RS512 are in `ValidMethods`. Symmetric algorithms are blocked.

**Rationale:**
- Asymmetric keys: private key signs (auth service only), public key verifies (any service)
- No shared secret risk — compromise of a verifying service doesn't compromise signing capability
- JWKS endpoint enables key rotation without redeploying all services
- Aligns with industry standard for inter-service JWT verification

**Trade-off:**
- RSA keys are larger and slower than HMAC
- Requires a JWKS-serving auth service (or static public key management)

---

## ADR-009: DB Alias Normalization Strategy

**Status:** Accepted (v1.0.0)

**Context:**
DB alias in env vars must be uppercase (e.g., `DB_TRANSACTION_HISTORY_DRIVER`), but Go maps are case-sensitive.

**Decision:**
`config.NormalizeDBAlias(name)` lowercases all DB aliases. All internal keying uses normalized form.

**Rationale:**
- Env var conventions use UPPERCASE
- Go idiomatic map keys use lowercase
- Normalization at a single point prevents scattered case handling
- `app.SQLByName("transaction_history")` always works regardless of env casing

**Trade-off:**
- Non-obvious behavior — must be documented (see `docs/CONFIGURATION_PROFILES.md`)
- Accidental alias collision if services use aliases that normalize to the same string

---

## ADR-010: Memcached Miss = Healthy

**Status:** Accepted (v1.0.0)

**Context:**
Readiness probes for cache could use GET (check miss vs. error) or SET/GET round-trips.

**Decision:**
Memcached `/ready` check uses GET on a sentinel key. Cache miss = healthy. Only network/timeout errors = unhealthy.

**Rationale:**
- A working Memcached that doesn't have a key is operating correctly
- Eliminates need for side-effect writes (SET) in readiness probes
- Network failures are the only meaningful signal for readiness
- Consistent with how caches actually behave in production

**Trade-off:**
- Misleading to operators unfamiliar with the convention — must be documented
