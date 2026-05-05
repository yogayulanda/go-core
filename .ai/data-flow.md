# go-core · Data Flow

## Request Lifecycle (Full Path)

```
Client (REST)                Client (gRPC)
     │                            │
     ▼                            │
┌────────────────────┐            │
│  HTTP Gateway      │            │
│  server/gateway/   │            │
│                    │            │
│  1. Request ID     │            │
│     inject         │            │
│  2. OTEL timer     │            │
│     start          │            │
│  3. Signature      │            │
│     validate       │            │
│     (if enabled)   │            │
│  4. Serialize →    │            │
│     protobuf       │            │
└────────┬───────────┘            │
         │                        │
         ▼ gRPC (internal)        ▼ gRPC (direct)
┌────────────────────────────────────────────────┐
│  gRPC Server                                   │
│  server/grpc/                                  │
│                                                │
│  Interceptors (in order):                      │
│  1. OTEL tracing                               │
│  2. Request ID propagation                     │
│  3. Auth: JWT verify or metadata extract       │
│     Claims → context                          │
│  4. Request metrics (counter + histogram)      │
│  5. grpc_request ServiceLog                    │
│  6. Panic recovery                             │
└────────────────────┬───────────────────────────┘
                     │
                     ▼
┌────────────────────────────────────────────────┐
│  Service Handler (consuming service)           │
│                                                │
│  Claims ← security.FromContext(ctx)            │
│  RequestID ← observability.RequestIDFromCtx    │
│                                                │
│  dbtx.WithTx(ctx, db, func(txCtx) error {      │
│      repo.Save(txCtx, data)                    │
│      outbox.PublishTx(txCtx, topic, payload)   │
│  })                                            │
│                                                │
│  logger.LogTransaction(ctx, tx)                │
└────────────────────────────────────────────────┘
```

---

## Observability Pipeline

```
Request arrives
     │
     ├─ OTEL span created (server interceptor)
     │   └─ propagated through context
     │
     ├─ Prometheus counter incremented (per request)
     │
     ├─ Handler executes
     │   ├─ DBLog emitted on each DB operation
     │   ├─ ServiceLog emitted on key operation events
     │   └─ TransactionLog emitted on business transaction completion
     │
     └─ Response sent
         ├─ OTEL span ended (with status)
         ├─ Prometheus histogram observed (latency)
         └─ grpc_request ServiceLog emitted (final status)

Prometheus scrape: GET /metrics
OTEL export: OTLP push to configured endpoint
```

---

## Startup Data Flow

```
main()
  │
  ├─ ENV variables read
  │   config.Load() → Config struct
  │   config.Validate() → fail fast on missing required
  │
  ├─ migration.AutoRunUpWithLogger() (if MIGRATION_AUTO_RUN=true)
  │   ├─ acquire distributed lock
  │   ├─ run pending Goose migrations
  │   └─ release lock
  │
  ├─ app.New()
  │   ├─ logger.New()           → emit nothing yet
  │   ├─ observability.NewMetrics() → register Prometheus metrics
  │   ├─ observability.InitTracing() → connect OTEL exporter
  │   ├─ database.New() × N     → open and health-check DB pools
  │   ├─ cache.NewRedis()       → open and ping Redis (if enabled)
  │   ├─ cache.NewMemcached()   → open and test Memcached (if enabled)
  │   └─ emit "app_init" ServiceLog
  │
  ├─ Service wires handlers
  │   ├─ build gRPC server with interceptors
  │   └─ build HTTP gateway with middleware
  │
  ├─ server.Run()
  │   ├─ bind ports
  │   ├─ server.LogStartupReadiness() → emit readiness ServiceLog
  │   └─ multiplex gRPC + HTTP
  │
  └─ Blocks on SIGINT/SIGTERM
      └─ lifecycle.Shutdown()
          ├─ DB pool close
          ├─ Redis close
          ├─ Kafka publisher/consumer close
          ├─ OTEL exporter flush
          └─ emit "app_runtime" shutdown ServiceLog
```

---

## Context Propagation Map

| Context Key | Set By | Read By | Content |
|---|---|---|---|
| Request ID | gRPC interceptor / gateway middleware | All layers | UUID string |
| OTEL Span | OTEL interceptor | `observability.FromContext` | Active trace span |
| Auth Claims | gRPC auth interceptor | Service handlers | `*security.Claims` |
| DB Transaction | `dbtx.Inject` | `dbtx.FromContext` | `*sql.Tx` |

All propagation is via standard `context.Context` — no global state.

---

## Log Flow

```
logger.Logger
    │
    ├─ LogService(ctx, ServiceLog)
    │   └─ zap.Info("service_log", fields...)
    │       ├─ category: "service"
    │       ├─ operation, status, duration_ms
    │       ├─ error_code (if present)
    │       └─ metadata (sanitized)
    │
    ├─ LogDB(ctx, DBLog)
    │   └─ zap.Info("db_log", fields...)
    │       ├─ category: "db"
    │       ├─ db_name, operation, table, status, rows_affected, duration_ms
    │       └─ error_code (if present)
    │
    └─ LogTransaction(ctx, TransactionLog)
        └─ zap.Info("transaction_log", fields...)
            ├─ category: "transaction"
            ├─ operation, transaction_id, user_id, status, duration_ms
            ├─ error_code (if present)
            └─ metadata (sanitized)

All fields pass through sanitizeFieldValue():
    sensitive keys → masked (last 2 chars shown)
```

---

## Outbox Data Flow

```
Service writes:
  dbtx.WithTx → {
    INSERT INTO domain_table ...   ← domain row
    INSERT INTO outbox_table ...   ← outbox row (topic, payload, status=pending)
  } COMMIT

OutboxWorker (explicit, service-controlled goroutine):
  loop every interval:
    SELECT * FROM outbox_table WHERE status='pending' LIMIT batch_size
    for each row:
      Kafka.Publish(row.topic, row.payload)
      if success: UPDATE status='published'
      if fail:    UPDATE status='failed', retry_count++
    emit outbox_batch ServiceLog
    emit app_outbox_batch_total counter
```

---

## Readiness Check Flow

```
GET /ready
  │
  ├─ For each required database:
  │   db.PingContext(ctx) → fail → 503
  │
  ├─ Redis (if enabled):
  │   client.Ping() → fail → 503
  │
  ├─ Memcached (if enabled):
  │   client.Get("__healthcheck__")
  │   → miss is OK (healthy)
  │   → network error → 503
  │
  └─ All pass → 200 {"status": "ok"}
```

---

## Error Response Data Flow

```
Handler returns AppError
    │
    ├─ [gRPC path]:
    │   errors.ToGRPC(err) → grpc.Status{Code, Message}
    │   Internal Err field → logged in ServiceLog, NOT in response
    │
    └─ [HTTP gateway path]:
        grpc-gateway translates gRPC status → HTTP JSON
        {
          "code": "UNAUTHORIZED",
          "message": "unauthorized request",
          "request_id": "uuid",
          "details": [...] ← only for validation errors
        }
        Deep technical errors → sanitized to "internal server error"
```
