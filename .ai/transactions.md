# go-core · Transactions

## Transaction Flow

`go-core` distinguishes two transaction concepts:

| Concept | Package | Purpose |
|---|---|---|
| **SQL Transaction** | `dbtx` | Database commit/rollback orchestration |
| **Business Transaction** | `logger.TransactionLog` | Observability for business-level operations |

These are **separate concerns** and must not be conflated.

---

## SQL Transaction Flow (`dbtx`)

### Pattern

```go
err := dbtx.WithTx(ctx, db, func(txCtx context.Context) error {
    // 1. Repository uses txCtx to get the active transaction
    if err := repo.SaveDomainData(txCtx, data); err != nil {
        return err // triggers automatic rollback
    }
    // 2. Write outbox record IN THE SAME TRANSACTION
    if err := outboxPublisher.PublishTx(txCtx, event); err != nil {
        return err // triggers automatic rollback
    }
    return nil // triggers commit
})
```

### Transaction Lifecycle

```
dbtx.WithTx(ctx, db, fn)
    │
    ├─ db.BeginTx(ctx, opts)           → START TRANSACTION
    ├─ Inject tx into context          → dbtx.Inject(ctx, tx)
    ├─ fn(txCtx)                       → execute business logic
    │   ├─ dbtx.FromContext(txCtx)    → repository retrieves tx
    │   └─ returns error?
    │       ├─ YES → tx.Rollback()    → ROLLBACK + return error
    │       └─ NO  → tx.Commit()     → COMMIT
    │
    └─ defer: panic recovery → Rollback + re-panic
```

### Repository Contract

```go
// Repository correctly uses context to get the active transaction
func (r *repo) Save(ctx context.Context, data Domain) error {
    db := dbtx.FromContext(ctx) // returns *sql.Tx if in transaction, else *sql.DB
    _, err := db.ExecContext(ctx, "INSERT INTO ...", data.Field)
    return err
}
```

### Rules

- **Start tx at the use-case boundary** — not inside repositories
- **Keep transactions short** — no outbound HTTP/gRPC calls inside a DB transaction
- **Repositories must not begin their own transactions** — use `dbtx.FromContext`
- **Outbox record must be in the same transaction** as domain data write
- **Rollback is automatic** on any error returned from `fn`
- **Panic recovery** is built-in — rollback happens on panic then re-panic

---

## Idempotency Strategy

`go-core` provides the **infrastructure but not the enforcement** of idempotency.

### Framework-Provided

- `dbtx.WithTx` enables atomic writes that underpin idempotency tables
- `cache` package (Redis/Memcached) can be used for idempotency key storage
- Outbox pattern ensures at-least-once delivery semantics with deduplication responsibility in the consumer

### Service-Owned Responsibility

Each consuming service must implement:

```
Idempotency Record Schema (recommended):
┌─────────────────────────────────────────────┐
│ idempotency_key  VARCHAR PK                 │
│ request_hash     VARCHAR (payload hash)     │
│ status           VARCHAR (pending/done/fail)│
│ result_snapshot  JSON (serialized response) │
│ expires_at       TIMESTAMP                  │
└─────────────────────────────────────────────┘
```

**Rule:** Same key + different payload → reject with `INVALID_REQUEST`.
**Rule:** Same key + same payload + status=done → return cached result.

### Missing: No Framework-Level Idempotency Key Middleware

`go-core` does NOT currently provide:
- gRPC/HTTP middleware to extract and validate idempotency keys
- Idempotency key storage helpers
- Cache-backed idempotency enforcement

This is a known gap — consuming services implement this independently.

---

## Failure Handling

### Transaction Failure Matrix

| Failure Point | Behavior | Recovery |
|---|---|---|
| `BeginTx` fails | Return error immediately — no retry in `dbtx` | Caller handles |
| `fn` returns error | Automatic `Rollback()` | Caller decides retry |
| `Commit` fails | Returns `"dbtx: commit failed"` error | Caller must handle — partial state possible |
| `Rollback` fails after fn error | Both errors joined via `errors.Join` | Log and alert |
| Panic inside `fn` | Rollback triggered, panic re-raised | Service-level panic handler |
| `db` is nil | Returns `"dbtx: db is nil"` immediately | Configuration error |

### Distributed Failure: Outbox Pattern

When a DB commit succeeds but Kafka publish fails:

```
WITHOUT outbox (direct publish):
  ✓ DB committed
  ✗ Kafka publish failed → event lost permanently

WITH outbox:
  ✓ DB committed (domain + outbox record in same tx)
  → Outbox worker picks up pending record
  → Retries Kafka publish with exponential backoff
  ✓ Event eventually delivered (at-least-once)
```

### Retry Strategy

`resilience.Do(ctx, opts, fn)` provides:
- **Exponential backoff** with configurable `BaseDelay`, `MaxDelay`
- **Crypto-random jitter** (`crypto/rand`) — avoids thundering herd
- **Retryable predicate** — services define what errors warrant retry
- **Context-aware** — respects `ctx.Done()` between attempts
- **OnRetry hook** — for logging/metrics on each retry event

Default settings:
```
MaxAttempts: 1
BaseDelay:   200ms
MaxDelay:    2s
Jitter:      100ms
Retryable:   context.DeadlineExceeded only
```

> **Note:** `DefaultRetryOptions().MaxAttempts = 1` means **no retry by default**.
> Services must explicitly set `MaxAttempts > 1` to enable retries.

### Circuit Breaker

`resilience.NewCircuitBreaker(opts)` wraps Sony Gobreaker:
- **Trips** after `ConsecutiveFailures > 5` (default)
- **Half-open timeout:** 60s
- **Context cancellation** is not counted as a failure
- Returns `ErrCircuitOpen` when tripped — maps to `SERVICE_UNAVAILABLE` in the error contract

---

## Business Transaction Observability

Use `logger.LogTransaction(ctx, tx)` for monitoring business-level flows.

```go
logger.LogTransaction(ctx, logger.TransactionLog{
    Operation:     "payment_process",       // stable operation name
    TransactionID: "TXN-20260213-0001",     // business correlation ID
    UserID:        "user_12345",            // actor; empty for system flows
    Status:        "failed",               // "success" | "failed" | "pending"
    DurationMs:    120,
    ErrorCode:     "PAYMENT_TIMEOUT",      // stable code for alerting
    Metadata: map[string]interface{}{
        "provider":    "bca",
        "channel":     "mobile_app",
        "amount":      150000,
    },
})
```

**Prometheus metric emitted:** `app_transaction_total{service, operation, status}`

### TransactionLog Rules

- `Operation` must be a **stable, snake_case name** — used in dashboards and alerts
- `TransactionID` is the **business identifier**, not the request ID
- `Status` must be one of: `success`, `failed`, `pending`
- `ErrorCode` must be **stable** — changing it breaks alert rules
- **Never put sensitive data** (tokens, amounts without business need, PII) in `Metadata`
- `dbtx` (SQL transaction) and `TransactionLog` (observability) are **separate concerns**

---

## Messaging Outbox: Transactional Delivery

```
Use Case: SQL write + Event publication must succeed together

Recommended flow:
  dbtx.WithTx(ctx, db, func(txCtx) error {
      repo.SaveOrder(txCtx, order)           ← domain write
      outbox.PublishTx(txCtx, "order.created", payload)  ← outbox write
      return nil                             ← single commit
  })

  OutboxWorker (explicit, service-controlled):
      loop:
          SELECT pending rows
          Kafka.Publish(row)
          UPDATE row status = "published"
```

- Worker is **never started automatically** by `go-core`
- Service controls: interval, batch size, publisher, which pod runs the worker
- Worker emits: `outbox_worker` ServiceLog + `app_outbox_batch_total` Prometheus metric
