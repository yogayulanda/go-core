# go-core · Modules

## Module Reference

### `app/`

**Purpose:** Bootstrap and runtime container. Wires all infrastructure dependencies together.

| Symbol | Description |
|---|---|
| `app.New(ctx, cfg)` | Initialize all infra (logger, metrics, DB, cache, tracing, lifecycle) |
| `app.App.Start(ctx)` | Block until context cancels, then trigger graceful shutdown |
| `app.App.SQLByName(name)` | Get DB pool by normalized alias |
| `app.App.SQLAll()` | Get all initialized DB pools |
| `app.App.RedisCache()` | Get Redis cache client (nil if disabled) |
| `app.App.MemcachedCache()` | Get Memcached client (nil if disabled) |
| `app.App.NewKafkaPublisher(opts...)` | Create publisher + register lifecycle close |
| `app.App.NewKafkaConsumer(...)` | Create consumer + register lifecycle close |
| `app.App.Lifecycle()` | Access lifecycle for custom shutdown hooks |
| `app.App.Logger()` | Access initialized logger |
| `app.App.Metrics()` | Access initialized Prometheus metrics |

**Important:** `app.New()` does NOT start Kafka consumers or outbox workers. Service decides.

---

### `config/`

**Purpose:** Environment-driven typed configuration with strict validation.

| Symbol | Description |
|---|---|
| `config.Load(ctx, cfg)` | Parse env vars into `*config.Config` struct |
| `cfg.Validate()` | Fail-fast validation — returns first error |
| `cfg.ValidateIssues()` | Structured validation — returns all issues |
| `config.NormalizeDBAlias(name)` | Lowercase normalization of DB alias keys |

**Key config structs:**
```
Config
├── AppConfig          SERVICE_NAME, ENVIRONMENT, LOG_LEVEL, SHUTDOWN_TIMEOUT
├── Databases          map[alias] → DBConfig (driver, DSN, pool settings, required)
├── GRPCConfig         GRPC_PORT, TLS settings
├── HTTPConfig         HTTP_PORT, TLS settings, pprof
├── ObservabilityConfig OTLP endpoint, sampling ratio
├── AuthConfig
│   ├── InternalJWTConfig  JWT enable/keys/issuer/audience/methods/leeway
│   └── SignatureConfig    HMAC signature key/headers/drift
├── RedisConfig        enable, address, password, DB
├── MemcachedConfig    enable, servers, timeout
└── KafkaConfig        enable, brokers, SASL, JKS
```

**DB alias normalization:** `DB_LIST=TRANSACTION_HISTORY` → env prefix `DB_TRANSACTION_HISTORY_*`
→ key normalized to `"transaction_history"` in `app.SQLByName("transaction_history")`

---

### `server/`

**Purpose:** gRPC and HTTP gateway orchestration, readiness, and health.

| Symbol | Description |
|---|---|
| `server.Run(...)` | Start gRPC + HTTP gateway, block, graceful shutdown |
| `server.LogStartupReadiness(...)` | Emit readiness log after bind |
| Standard endpoints | `GET /ready`, `GET /health`, `GET /metrics` |

**gRPC interceptors (always active):**
- OTEL tracing
- Request ID injection
- Auth extraction/verification (JWT or metadata)
- Request metrics (`app_request_total`, `app_request_duration_seconds`)
- Service metrics via `ServiceLog`
- Panic recovery + sanitized error response

**Gateway middleware (always active):**
- Request ID injection
- HTTP panic recovery
- HMAC signature validation (if enabled)
- HTTP metrics (`app_http_request_total`, `app_http_request_duration_seconds`)

---

### `security/`

**Purpose:** JWT verification and claims extraction.

| Symbol | Description |
|---|---|
| `security.NewInternalJWTVerifier(cfg)` | Build verifier from config (JWKS or static key) |
| `verifier.Verify(token)` | Validate token, return `*Claims` or error |
| `verifier.ShouldAuthenticate(method)` | Check include/exclude method policy |
| `verifier.AuthMode()` | Returns `"jwt"` or `"metadata"` |
| `verifier.ConfigMetadata()` | Returns startup diagnostics map |
| `security.ExtractFromMetadata(ctx)` | Extract claims from gRPC metadata headers |
| `security.AuthErrorCode(err)` | Stable error code string for logging |

See `.ai/security.md` for full auth flow.

---

### `errors/`

**Purpose:** Canonical error contract and transport mapping.

| Symbol | Description |
|---|---|
| `errors.AppError` | Canonical error type: `Code`, `Message`, `Category`, `Details`, `Err` |
| `errors.New(code, msg)` | Create AppError without wrapping |
| `errors.Wrap(code, msg, err)` | Create AppError wrapping internal error |
| `errors.Validation(msg, details...)` | Create validation error with field details |
| `errors.ToGRPC(err)` | Map AppError → gRPC status code |

**Error code → gRPC mapping:**
```
INVALID_REQUEST     → InvalidArgument
UNAUTHORIZED        → Unauthenticated
FORBIDDEN           → PermissionDenied
NOT_FOUND           → NotFound
SESSION_EXPIRED     → Unauthenticated
SERVICE_UNAVAILABLE → Unavailable
INTERNAL_ERROR      → Internal
```

**Critical rule:** `Err` field (internal error) is NEVER exposed to clients — stays in logs only.

---

### `logger/`

**Purpose:** Structured Zap-based logging with redaction and distinct log flavors.

| Symbol | Description |
|---|---|
| `logger.New(service, level)` | Create logger instance |
| `logger.Logger.LogService(ctx, log)` | Service-flow log → `service_log` |
| `logger.Logger.LogDB(ctx, log)` | Database operational log → `db_log` |
| `logger.Logger.LogTransaction(ctx, log)` | Business transaction log → `transaction_log` |
| `logger.Logger.Info/Warn/Error(ctx, msg, fields...)` | Generic structured log |
| `logger.Logger.WithComponent(name)` | Create child logger with component tag |

**Log flavors:**
- `ServiceLog` — standard service operation flow (use by default)
- `DBLog` — DB query/operation diagnostics
- `TransactionLog` — platform-standard for transaction-oriented services only

**Redaction:** Automatic on any sensitive key in any log field map. See `.ai/security.md`.

---

### `observability/`

**Purpose:** Prometheus metrics registry and OTEL tracing bootstrap.

**Key metrics (stable — do not change names):**

| Metric | Labels | Purpose |
|---|---|---|
| `app_request_total` | service, method, status | gRPC request count |
| `app_request_duration_seconds` | service, method | gRPC latency |
| `app_http_request_total` | service, method, route, status | HTTP request count |
| `app_http_request_duration_seconds` | service, method, route | HTTP latency |
| `app_service_operation_total` | service, operation, status | Service-level ops |
| `app_service_operation_duration_seconds` | service, operation | Service-level latency |
| `app_db_operation_total` | service, db_name, operation, status | DB ops |
| `app_db_operation_duration_seconds` | service, db_name, operation | DB latency |
| `app_message_publish_total` | service, topic, status | Kafka publishes |
| `app_message_consume_total` | service, topic, group, status | Kafka consumes |
| `app_message_process_duration_seconds` | service, topic, group | Consumer latency |
| `app_outbox_batch_total` | service, status | Outbox batch runs |
| `app_outbox_batch_duration_seconds` | service | Outbox batch latency |
| `app_outbox_batch_size` | service | Outbox batch distribution |
| `app_transaction_total` | service, operation, status | Business transactions |

**Important:** Metrics are registered as global singletons via `sync.Once`. Do not call `NewMetrics()` more than once per process.

---

### `dbtx/`

**Purpose:** SQL transaction orchestration and context propagation.

| Symbol | Description |
|---|---|
| `dbtx.WithTx(ctx, db, fn)` | Execute fn in a transaction (default options) |
| `dbtx.WithTxOptions(ctx, db, opts, fn)` | Execute fn with custom `*sql.TxOptions` |
| `dbtx.Inject(ctx, tx)` | Store `*sql.Tx` in context |
| `dbtx.FromContext(ctx)` | Retrieve `*sql.Tx` from context (or fallback to `*sql.DB`) |

See `.ai/transactions.md` for full transaction flow.

---

### `messaging/`

**Purpose:** Kafka producer, consumer, and transactional outbox abstractions.

| Symbol | Description |
|---|---|
| `messaging.NewKafkaPublisher(cfg, opts...)` | Create Kafka producer |
| `messaging.NewKafkaConsumer(cfg, topic, group, handler, opts...)` | Create consumer |
| `messaging.Publisher.Publish(ctx, msg)` | Publish message directly |
| `messaging.Consumer.Start(ctx)` | Start consuming |
| `outbox.NewWorker(repo, publisher, opts...)` | Create outbox worker |
| `outbox.Worker.StartChecked(ctx)` | Start worker loop (service-owned goroutine) |
| `outbox.Worker.RunOnce(ctx)` | Process one batch (for tests/admin jobs) |
| `outbox.NewPublisher(repo)` | Write outbox records in transaction |
| `outbox.Publisher.PublishTx(ctx, topic, payload)` | Write outbox row (must be inside dbtx) |

**Outbox delivery guarantee:** At-least-once. Consumer must handle duplicate detection.
**Worker is never auto-started by `go-core`.** Service decides process topology.

---

### `resilience/`

**Purpose:** Retry with exponential backoff, circuit breaker, and timeout.

| Symbol | Description |
|---|---|
| `resilience.Do(ctx, opts, fn)` | Execute fn with retry and backoff |
| `resilience.DefaultRetryOptions()` | Baseline options (MaxAttempts=1, no retry) |
| `resilience.NewCircuitBreaker(opts)` | Sony Gobreaker wrapper |
| `resilience.DefaultCircuitBreakerOptions(name)` | Baseline CB (trips at 5 consecutive failures) |
| `resilience.WithTimeout(ctx, d, fn)` | Execute fn with deadline |
| `resilience.IsTransientError(err)` | Default retryable predicate |

**Default retry = 1 attempt = no retry.** Set `MaxAttempts > 1` explicitly.

---

### `cache/`

**Purpose:** Redis and Memcached adapters with unified `Cache` interface.

| Symbol | Description |
|---|---|
| `cache.NewRedisFromConfig(cfg, logger)` | Create Redis client |
| `cache.NewMemcachedFromConfig(cfg, logger)` | Create Memcached client |
| `cache.Cache` | Interface: `Get`, `Set`, `Delete`, `Close` |

**Readiness behavior:**
- Redis: ping check at startup and in `/ready`
- Memcached: cache-miss is **healthy**; network error is unhealthy

---

### `migration/`

**Purpose:** Goose-based schema migration with distributed locking.

| Symbol | Description |
|---|---|
| `migration.AutoRunUp(ctx, db, cfg)` | Run pending migrations at startup |
| `migration.AutoRunUpWithLogger(ctx, db, cfg, log)` | Same with structured startup logging |

**Distributed lock:** Prevents concurrent migration in multi-pod K8s deployments.
Lock is implemented per DB driver: `sp_getapplock` (MSSQL), `GET_LOCK` (MySQL), advisory lock (Postgres).

---

### `httpclient/`

**Purpose:** Resilient outbound HTTP client — wraps Resty with circuit breaker, retry, OTEL tracing, and structured logging. Available from **v1.1.0**.

| Symbol | Description |
|---|---|
| `httpclient.NewClient(log, opts...)` | Create HTTP client with configured resilience |
| `client.Do(ctx, req, method, url)` | Execute request inside circuit breaker |
| `client.Get(ctx, url)` | Convenience GET |
| `client.Post(ctx, url, body)` | Convenience POST |
| `client.Request()` | Returns a raw `*resty.Request` for full control |

**Options:**
```go
httpclient.WithTimeout(5 * time.Second)
httpclient.WithRetry(&resilience.RetryOptions{MaxAttempts: 3, ...})
httpclient.WithCircuitBreaker(&resilience.CircuitBreakerOptions{...})
httpclient.WithTracing(true)
httpclient.WithUserAgent("my-service/1.0")
```

**Behavior:**
- Emits `httpclient_request` ServiceLog on every outbound call (before)
- Emits `httpclient_response` ServiceLog on every response (after), with `status_code` and `duration_ms`
- HTTP 5xx responses **trigger the circuit breaker** counter
- HTTP 429 and 5xx trigger retry (when retry is configured)
- OTEL trace context is propagated via `otelhttp` transport

**Rule:** Use `httpclient` for all outbound service-to-service HTTP calls — not `http.DefaultClient`.

---

### `version/`

**Purpose:** Build metadata injection.

| Symbol | Description |
|---|---|
| `version.Version` | Semver string |
| `version.Commit` | Git commit SHA |
| `version.BuildDate` | Build timestamp |

Must be set via `ldflags` at build time. Emitted at startup in `app_init` log.
