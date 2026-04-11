# PROJECT CONTEXT FILE (PCF)

## 1. Project Identity
- **Project Name**: go-core
- **Business Domain**: Infrastructure / Foundational Framework
- **Primary Purpose**: Provide a domain-agnostic, reusable infrastructure wrapper and runtime foundation for Go microservices (e.g., graceful shutdown, transport routing, logging, database initialization).
- **System Status**: Production (Stable v1.0.0 releases)
- **System Criticality**: Mission Critical (Forms the bedrock for all consuming Go microservices).

## 2. Scope and System Boundaries
### 2.1 In Scope
- App container lifecycle and graceful shutdown management.
- Configuration loading and strict validation from Environment Variables.
- Structured contextual logging (Zap) featuring JSON, sensitive field masking, and distinct output flavors (`ServiceLog`, `DBLog`, `TransactionLog`).
- Multi-database (SQLServer, MySQL, PostgreSQL) initialization and runtime connection pooling.
- Cache initialization (Redis, Memcached) with configured fail-fast readiness probes.
- Messaging abstractions (Kafka Producer/Consumer, Outbox patterns).
- OpenTelemetry observability bootstrap and extensive Prometheus metric exposure (`/metrics`).
- Resilience patterns (Timeout, Sony gobreaker Circuit Breakers).
- Extensible Security handlers (JWT Validation, HTTP Payload Signatures).
- Common technical error wrappers and transport-layer masking.

### 2.2 Out of Scope
- Domain logic or business rules.
- Service schema semantics and API model logic.
- Product-specific event payload handling.
- Generic application helper utilities (which belong in `utils-shared`).

### 2.3 Upstream Dependencies
*As a framework, its runtime dependencies strictly mirror what the importing Microservice configures targetting:*
- Relational Databases (MSSQL, MySQL, Postgres)
- Redis / Memcached Instances
- Kafka Brokers
- OpenTelemetry Collectors (`OTEL_EXPORTER_OTLP_ENDPOINT`)
- Internal JWKS Servers for dynamic token keys

### 2.4 Downstream Consumers
- All Go microservices built in the ecosystem (e.g. `transaction-history-service`).

## 3. Stakeholders and Users
- **End Users**: Platform Engineers, Backend Software Engineers building microservices.
- **Internal Users / Ops**: DevOps and SRE utilizing the standardized Prometheus metrics and uniform Zap JSON logs over ELK/Datadog.
- **System Owner**: Platform / Architecture Team.

## 4. High-Level Architecture
### 4.1 Core Components
- **`config`**: Environment parser with strict `.Validate()` requirements (`viper` / manual struct mapping).
- **`app`**: Orchestration context handling OS Interrupts and connection lifetimes.
- **`server/grpc` + `server/gateway`**: Uses standard `grpc-ecosystem/grpc-gateway`. Handles local multiplexing, registering standard API endpoints (`/ready`, `/health`, `/metrics`), and binds global interceptors (OTEL tracing, JWT extraction/enforcement).
- **`database` & `dbtx`**: Abstractions around database clients allowing robust context-based transaction propagation natively over standard `context.Context`.
- **`security`**: `MicahParks/keyfunc/v3` driven RS256/384/512 token parsing and evaluation. Optional HTTP Payload HMAC verification.

### 4.2 Architecture Style
- Decoupled Library / Modular Foundation Module.
- 12-factor app compliant.

## 5. Primary Technical Flows
*As this is an infrastructure library, flows apply to startup instrumentation and request routing, rather than business logic.*

### Flow: Service Bootstrap (Golden Path)
1. **Trigger**: Microservice executes `main()`.
2. **Context Setup**: Traps SIGINT/SIGTERM (`signal.NotifyContext`).
3. **Execution**:
   - `config.Load(...)` parses `ENV` or `.env`.
   - Explicit `cfg.Validate()` aborts startup if constraints miss.
   - Initializer routines check DB Connections, execute Auto Migrations via `Goose` if instructed.
   - `app.New(ctx, cfg)` bonds caches and messaging abstractions.
   - Network transports (gRPC, gRPC-Gateway) are created and attached to handlers.
   - `server.LogStartupReadiness` spins up.
   - Blocks on `server.Run(...)` which manages multiplexing and listening over specified ports.
4. **Conclusion**: On SIGINT, unwinds connections via established `SHUTDOWN_TIMEOUT` ruleset returning cleanly.

## 6. Technical Execution Flow (Request Routing)
Incoming request mapped from client to gRPC:
1. `GET`/`POST` REST request arrives on `HTTP_PORT`.
2. `grpc-gateway` receives request, initiates OTEL metric timer and attaches request IDs.
3. HTTP signature validation occurs (if `AUTH_SIGNATURE_ENABLED=true`).
4. `grpc-gateway` serializes to Protobuf and forwards internally.
5. Internal gRPC network invokes auth interceptors: Validates JWT Bearer Token (if `INTERNAL_JWT_ENABLED=true`) parsing Subject, Roles, Attributes.
6. Execution transitions downstream into the Microservice logic layer via the registered handler.
7. On handler finish or crash, `server/grpc` traps panics, formats generic standard Error contracts masking deep technical secrets, writes to ELK via `ServiceLog`.

## 7. Data and State Management
- Completely stateless. Operates functionally using passed configuration pointers.
- Provides standard `DBLog` schema which consumer apps use to standardize query durations/failures to logging.

## 8. Security and Compliance
- **Authentication mechanism**: Highly configurable JWT validation natively intercepting GRPC requests. Validates time constraints (`nbf`, `exp`, `iat`) and Audience/Issuer configurations smoothly tracking dynamic keys from `INTERNAL_JWT_JWKS_ENDPOINT`.
- **Authorization rules**: Maps validated tokens to `security.Claims` structurally injected into Context for use in Handlers.
- **Security policies**: Specific Full gRPC Paths can be exclusively included or excluded from Auth requirements (`INTERNAL_JWT_INCLUDE_METHODS`, `INTERNAL_JWT_EXCLUDE_METHODS`).
- **Telemetry Compliance**: Sensitive variable logs mask all but the last 2 characters inherently within the Logger definitions.

## 9. Operations and Production Behavior
- **Readiness API**: Uniform `GET /ready` natively exposed by Gateway. Polls Database/Kafka/Redis/Memcached if they are enabled in ENV checking live availability. Will hard-shift to HTTP 503 during failover logic.
- **Fail-Fast Initialization**: If Redis/Memcached/DB are checked as `ENABLED` or `REQUIRED`, the orchestrator immediately crashes at `app.New()`.
- **Metrics**: Massive range of standard Prometheus metrics out-of-the-box (`app_http_request_total`, `app_db_operation_duration_seconds`, etc).

## 10. Constraints and Technical Debt
### 10.1 Constraints
- Imposes an inherently strict layout (requires using grpc-gateway rather than Gin/Fiber/Echo HTTP stacks).
- Imposes 12-factor Env configs entirely ruling out structured file configurations (YAML/JSON) directly driving business initialization.
- Relies extremely heavily on Context mapping (`dbtx`, `claims`, `request_id`).

## 11. Failure Modes and Edge Cases
- **Concurrent Migrations**: If `MIGRATION_AUTO_RUN` is active in a multi-pod K8s deployment environment, `go-core` gracefully executes database specific native locking queries (e.g. `sp_getapplock` in SQLServer or `GET_LOCK` in MySQL) to naturally pause and prevent concurrent corrupted Goose schema updates.
- **Cache Connection failures**: If an external Cache or Database dies *after* application startup, `go-core` circuit breakers via resilience packages intercept traffic gracefully. The `/ready` endpoint continuously shifts status exposing internal disruptions to K8s probes.

## 12. Decision History (Why the system is like this)
- **Decision**: Centralized and highly opinionated Error Contract (`errors.AppError`).
  - **Reasoning**: Protects API gateways from leaking deep database internals / SQL injection metadata inadvertently, enforcing standard UI/Frontend parsing capabilities on errors.
- **Decision**: Memcached considers a "cache miss" an explicitly healthy status for its `/ready` probe tests.
  - **Reasoning**: Caches missing keys indicates working networks and protocols. Fails only on network IO failures.

## 13. Rare, Implicit, or Non-Obvious Knowledge
- **DB Alias Normalization**: The ENV Configuration `DB_LIST=TRANSACTION_HISTORY` is required entirely in uppercase parameters (e.g. `DB_TRANSACTION_HISTORY_DRIVER="mysql"`). However, the resulting connection object during mapping is silently normalized down to `transaction_history` lowercase during internal keying.
- **HTTP/JSON Errors masking**: Internal classifications / HTTP failures bypass standard stack tracing, sanitizing external metadata locally into `ErrorInfo.reason` GRPC strings.
- Outbox workers must be started explicitly using `outbox.Worker.StartChecked(ctx)` even if registered, allowing consuming applications to decide *which* worker pods execute polling procedures.

## 14. Open Questions and Unknowns
- Testing and Unit Test mocking abstraction standard (e.g., whether `go-sqlmock` enforces behavior downstream over basic DB definitions).
- How generic OpenTelemetry custom attributes are merged back into `service_operation` metrics automatically over context.

## 15. Instructions for Future AI Usage
When using this Project Context File:
- Treat this document as the exclusive defining law for generic Go microservices under this infrastructure tree.
- When generating subsequent Go app components relying on `go-core`, default to GRPC bindings, Viper/Env struct loaders, and Zap structural telemetry, completely rejecting un-aligned alternatives.
- Do not ask questions already answered in this document.
