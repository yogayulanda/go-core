# go-core · Integrations

## External Service Dependencies

All integrations are **opt-in** — only initialized when explicitly configured.
`go-core` handles connection lifecycle (open, health check, graceful close).
Business logic for each integration lives in the consuming service.

---

## Databases (SQL)

**Supported drivers:** `sqlserver` (MSSQL), `mysql`, `postgres`

**Configuration:**
```env
DB_LIST=PRIMARY,READONLY          # comma-separated alias list (UPPERCASE)
DB_PRIMARY_DRIVER=postgres
DB_PRIMARY_HOST=db.host
DB_PRIMARY_PORT=5432
DB_PRIMARY_NAME=mydb
DB_PRIMARY_USER=user
DB_PRIMARY_PASSWORD=pass
DB_PRIMARY_REQUIRED=true          # fail-fast if unavailable
DB_PRIMARY_MAX_OPEN_CONNS=25
DB_PRIMARY_MAX_IDLE_CONNS=5
DB_PRIMARY_CONN_MAX_LIFETIME=5m
DB_PRIMARY_CONN_MAX_IDLE_TIME=1m
```

**Access pattern:**
```go
db := app.SQLByName("primary")     // normalized lowercase alias
```

**Lifecycle:** `app.New()` opens and health-checks pools; `lifecycle.Shutdown()` closes all pools.

**Migration:** Uses [Goose](https://github.com/pressly/goose) with distributed lock.
```env
MIGRATION_AUTO_RUN=true
MIGRATION_DB_NAME=primary
MIGRATION_DIR=./migrations
MIGRATION_LOCK_ENABLED=true
MIGRATION_LOCK_TIMEOUT=60s
```

---

## Redis

**Client library:** `go-redis/redis/v9`

**Configuration:**
```env
REDIS_ENABLED=true
REDIS_ADDRESS=redis:6379
REDIS_PASSWORD=secret
REDIS_DB=0
```

**Access pattern:**
```go
redis := app.RedisCache()          // nil if not enabled
```

**Readiness:** Ping check on startup + in `/ready`. Redis enabled = treated as required dependency.

---

## Memcached

**Client library:** `bradfitz/gomemcache`

**Configuration:**
```env
MEMCACHED_ENABLED=true
MEMCACHED_SERVERS=mc1:11211,mc2:11211
MEMCACHED_TIMEOUT=500ms
```

**Access pattern:**
```go
mc := app.MemcachedCache()         // nil if not enabled
```

**Readiness note:** Cache miss is **intentionally healthy** — only network/timeout failures trigger 503.

---

## Kafka

**Client library:** `IBM/sarama` (wrapped)

**Configuration:**
```env
KAFKA_ENABLED=true
KAFKA_BROKERS=kafka:9092,kafka2:9092
KAFKA_CLIENT_ID=my-service
KAFKA_USERNAME=user              # SASL plain
KAFKA_PASSWORD=pass
KAFKA_JKS_FILE=/path/cert.jks   # TLS via JKS
KAFKA_JKS_PASSWORD=keystorepass
```

**Publisher:**
```go
pub, err := app.NewKafkaPublisher(
    messaging.WithRetry(...),
    messaging.WithDLQ("my-service.dlq"),
    messaging.WithSuccessLog(true),
)
```

**Consumer:**
```go
consumer, err := app.NewKafkaConsumer(
    "my.topic", "my-consumer-group", handler,
    messaging.WithConcurrency(4),
    messaging.WithRetry(...),
)
```

**Lifecycle:** Publisher and consumer lifecycle automatically registered via `app.NewKafka*`.
**Outbox:** Requires explicit `outbox.NewWorker(...)` and `worker.StartChecked(ctx)` in service.

---

## OpenTelemetry (OTLP)

**SDK:** `go.opentelemetry.io/otel`

**Configuration:**
```env
OTLP_ENDPOINT=otel-collector:4317
OTLP_INSECURE=true               # disable TLS for internal collectors
OTLP_CA_CERT_FILE=/path/ca.crt   # TLS CA cert (if not insecure)
TRACE_SAMPLING_RATIO=0.1         # 10% sampling in production
```

**Automatic:** Traces are injected at gRPC interceptor level — services don't need manual span creation for transport.
**Manual spans:** Use `go.opentelemetry.io/otel/trace` directly in service code.
**Exporter:** OTLP gRPC push to configured collector endpoint.
**Shutdown:** Registered in lifecycle — flushes pending spans on graceful shutdown.

---

## JWKS Server (JWT Key Provider)

**Client library:** `MicahParks/keyfunc/v3`

**Configuration:**
```env
INTERNAL_JWT_ENABLED=true
INTERNAL_JWT_JWKS_ENDPOINT=https://auth.internal/jwks
INTERNAL_JWT_JWKS_REFRESH_INTERVAL=5m
INTERNAL_JWT_ISSUER=https://auth.internal
INTERNAL_JWT_AUDIENCE=my-service
```

**Behavior:** Keys are fetched and cached at startup. Background refresh at configured interval.
**Fallback:** If JWKS endpoint is empty, static `INTERNAL_JWT_PUBLIC_KEY` PEM is used.

---

## Go Module Dependencies

Key external dependencies (from `go.mod`):

| Dependency | Purpose |
|---|---|
| `github.com/golang-jwt/jwt/v5` | JWT parsing and validation |
| `github.com/MicahParks/keyfunc/v3` | JWKS key fetching and caching |
| `github.com/IBM/sarama` | Kafka producer/consumer |
| `github.com/redis/go-redis/v9` | Redis client |
| `github.com/bradfitz/gomemcache` | Memcached client |
| `github.com/pressly/goose/v3` | Schema migration |
| `go.opentelemetry.io/otel` | Distributed tracing |
| `github.com/prometheus/client_golang` | Prometheus metrics |
| `go.uber.org/zap` | Structured logging |
| `github.com/sony/gobreaker/v2` | Circuit breaker |
| `github.com/grpc-ecosystem/grpc-gateway/v2` | HTTP-to-gRPC gateway |
| `google.golang.org/grpc` | gRPC transport |

**Go version target:** `1.24`

---

## No-Dependency Rule

`go-core` must NOT depend on:
- Any consuming service's domain package
- Any product-specific library or SDK
- Any business-logic framework (no ORMs, no DI containers, no event buses)

New external dependencies require explicit justification — keep the dependency surface minimal.
