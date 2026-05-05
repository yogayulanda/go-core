# go-core · Architecture

## System Design

`go-core` is a **framework/library repository** — not a runnable service.
It provides the runtime foundation that consuming microservices compose and own.

### Architectural Style

- **Modular infrastructure library** — each package is independently adoptable
- **12-Factor App compliant** — all config from environment variables
- **Explicit lifecycle** — no hidden background goroutines; consuming service registers all shutdown hooks
- **Opt-in infra** — Redis, Kafka, Memcached, tracing are never started unless configured

---

## Layer Model

```
┌─────────────────────────────────────────────────────┐
│                   Consuming Service                  │
│     (business logic, domain, handlers, proto)        │
└───────────────────────┬─────────────────────────────┘
                        │ uses
┌───────────────────────▼─────────────────────────────┐
│                      go-core                         │
│                                                      │
│  ┌─────────────────────────────────────────────┐    │
│  │  Transport Layer                             │    │
│  │  server/grpc  ·  server/gateway              │    │
│  │  (auth interceptors, metrics, panic recovery)│    │
│  └──────────────────┬──────────────────────────┘    │
│                     │                                │
│  ┌──────────────────▼──────────────────────────┐    │
│  │  Application Container (app/)               │    │
│  │  lifecycle · dependency wiring · shutdown    │    │
│  └──────────────────┬──────────────────────────┘    │
│                     │                                │
│  ┌──────────────────▼──────────────────────────┐    │
│  │  Infrastructure Layer                        │    │
│  │  database · dbtx · cache · messaging         │    │
│  │  migration · resilience                      │    │
│  └──────────────────┬──────────────────────────┘    │
│                     │                                │
│  ┌──────────────────▼──────────────────────────┐    │
│  │  Cross-Cutting Concerns                      │    │
│  │  logger · observability · security · errors  │    │
│  └─────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────┘
```

---

## Stable Bootstrap Path

```
main()
  │
  ├─ signal.NotifyContext(SIGINT, SIGTERM)
  │
  ├─ config.Load(ctx, cfg)        ← env parsing + normalization
  ├─ cfg.Validate()               ← fail-fast on missing required fields
  │
  ├─ migration.AutoRunUp(...)     ← optional, with distributed lock
  │
  ├─ app.New(ctx, cfg)            ← wire: logger, metrics, tracing,
  │                                  DB pools, Redis, Memcached, lifecycle
  │
  ├─ build gRPC server + register handlers
  ├─ build HTTP gateway + register handlers
  │
  └─ server.Run(...)              ← multiplex gRPC+HTTP, block on signal,
                                    graceful shutdown via lifecycle
```

---

## Service Interactions

### Transport Boundary

```
HTTP Client
    │ REST/JSON
    ▼
server/gateway (HTTP)
    │ HMAC signature check (optional)
    │ request-id injection
    │ OTEL metrics + tracing
    │ panic recovery
    │ translates → gRPC-Gateway → protobuf
    ▼
server/grpc (gRPC)
    │ JWT extraction + verification
    │ Claims injected into context
    │ grpc_request ServiceLog
    │ request metrics
    ▼
Service Handler (consuming service)
    │
    ├─ dbtx.WithTx → repository → dbtx.FromContext
    ├─ outbox.PublishTx (same transaction)
    └─ logger.LogTransaction / LogService
```

### Readiness Path

```
GET /ready
    │
    ├─ DB ping (required databases)
    ├─ Redis ping (if enabled)
    ├─ Memcached ping (if enabled)
    └─ 200 OK or 503 Service Unavailable
```

---

## High-Risk Change Areas

Changes in these packages must be treated as **public contract risk** first:

| Package | Risk |
|---|---|
| `app/` | Breaks every service bootstrap path |
| `config/` | Breaks every env-based configuration |
| `server/` | Breaks transport contract, readiness, health |
| `migration/` | Breaks migration autorun and lock semantics |
| `errors/` | Breaks transport-facing error contract |
| `security/` | Breaks cross-service auth behavior |
| `logger/` | Breaks operational observability baselines |
| `observability/` | Breaks Prometheus metric naming/labels |

---

## Scaling Considerations

- **Stateless by design** — `go-core` holds no mutable runtime state beyond initialized connections
- **Horizontal scaling**: All instances share the same connection pool config; connection counts multiply with replicas
- **Migration locking**: `MIGRATION_LOCK_ENABLED` prevents concurrent migration corruption in multi-pod K8s deployments
- **Outbox workers**: Worker pods vs. handler pods can be separated — consuming service controls topology
- **Metrics cardinality**: Avoid high-cardinality label values (e.g., user IDs) — label sets are defined at framework level and are intentionally bounded
- **Circuit breaker**: Per-instance state — no distributed state sharing; each pod has its own breaker counts

---

## Key Design Decisions

- **gRPC-gateway over Gin/Fiber/Echo** — enforces single protobuf contract, HTTP is a projection
- **Viper/env config over YAML** — enforces 12-factor compliance, simplifies container deployments
- **Context-based transaction propagation** — `dbtx` avoids passing `*sql.Tx` through function signatures
- **Explicit lifecycle ownership** — no hidden goroutines; consuming service decides what starts and when
- **Outbox pattern for event durability** — direct publish is lossy on crash; outbox ensures transactional delivery
