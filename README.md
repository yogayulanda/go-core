# go-core

`go-core` is a runtime foundation for Go microservices. It provides a gRPC server, an HTTP/gRPC gateway, configuration, logging, error mapping, messaging, databases, and observability. It handles common infrastructure work so each service does not need to rebuild it. It does not contain business entities, workflows, or product rules; those stay in your service.

## Core Concepts In 2 Minutes

- **Envelope:** a JSON wrapper around a successful API response. It gives clients one predictable place for status, code, message, request ID, timestamp, and data.
- **Response code:** a stable string such as `CORE-0000` or `CORE-1000`. The registry maps it to an HTTP/gRPC status and a safe message.
- **Request ID:** an identifier for one request. Logs and downstream calls can use it to connect related work.
- **Field bag:** request-scoped storage for the request ID, user ID, client IP, and registered labels. The gateway and gRPC interceptor create it for incoming requests.
- **Outbox:** a database record written in the same transaction as a business change. A worker publishes it later so a process stop does not silently lose the event.
- **Raw handler:** a route that opts out of the success envelope. Use it only when your handler must own the complete HTTP response.

Module: `github.com/yogayulanda/go-core`

## Requirements

- Go **1.26.0 or newer**. The module directive is the minimum required by the dependency graph. CI currently tests with Go 1.26.9.
- A supported SQL database, when persistence is needed: SQL Server or MySQL.
- Optional services: Kafka, Redis, Memcached, and an OTLP-compatible collector/APM endpoint.
- `make` and golangci-lint v2.14.0 for the full local development checks.

## Quick Start

Install the released module:

```sh
go get github.com/yogayulanda/go-core@v1.2.0
```

The complete, buildable service example is [`examples/quickstart/main.go`](examples/quickstart/main.go). It loads configuration, overrides it in code, creates the app, uses the gRPC Health service registered by `server/grpc.New`, registers an HTTP `/ping` route that calls gRPC, and runs both servers. This shows a real gateway-to-gRPC request without requiring generated protobuf files.

Create a local `.env`:

```dotenv
SERVICE_NAME=catalog-api
APP_ENV=local
LOG_LEVEL=info
GRPC_PORT=50051
HTTP_PORT=8080
METRICS_ENABLED=false
DIAG_ENABLED=true
```

Run and call the gateway:

```sh
go run ./examples/quickstart
curl -i http://localhost:8080/ping
```

The response is a go-core success envelope. `data` contains the handler JSON:

```json
{"status":"SUCCESS","response_code":"CORE-0000","message":"success","request_id":"...","timestamp":"...","data":{"message":"pong"}}
```

The generated request ID changes on each call. For the complete service, use the compiled example file; no partial bootstrap snippet is required.

## Request Flow

These are the real construction orders in `server/gateway.New` and `server/grpc.New`. Follow each chain from top to bottom: HTTP requests use the HTTP chain, while RPC calls use the gRPC unary chain. The gRPC OpenTelemetry stats handler runs alongside the unary chain.

```text
HTTP request
  -> request identity
  -> panic recovery
  -> request body limit
  -> request signature validation (when enabled)
  -> OpenTelemetry HTTP tracing
  -> CORS
  -> success envelope / Raw opt-out
  -> HTTP metrics
  -> grpc-gateway ServeMux and route handler

gRPC unary request
  -> panic recovery
  -> request ID and incoming identity restoration
  -> in-flight tracking
  -> access logging and application error mapping
  -> internal JWT / metadata auth
  -> gRPC metrics
  -> service handler
```

Each HTTP step has one job: identity creates/propagates request fields; recovery converts panics; body-limit rejects oversized input; signature checks the configured HMAC; tracing creates the server span; CORS applies origin policy; envelope wraps normal success responses; metrics records route/status; mux selects the registered handler.

Each gRPC step has one job: recovery catches panics; request-ID restores trusted incoming identity; in-flight tracking supports draining; logging maps errors and writes one access log; auth validates configured JWT/metadata; metrics records enabled counters/histograms; the service handler runs business code.

The gateway envelope is inside CORS and tracing. The outer identity, recovery, body-limit, and signature layers still apply to routes registered with `gateway.Raw`.

## Features

### Response Envelope

| | Guidance |
|---|---|
| What | A standard JSON wrapper around successful API data. |
| When to use | Normal service APIs that need consistent response metadata. |
| How to turn on | Default for non-operational gateway routes. |
| How to turn off | Wrap only the selected handler with `gateway.Raw` or `gateway.RawHTTP`. |

A response envelope is a standard JSON object around a successful API payload. It gives clients consistent status, error code, message, request ID, and timestamp fields. The gateway wraps 2xx responses; a registered response code supplies the envelope class/message and HTTP status.

Non-2xx status/body pass through, but response-code metadata headers are removed. `/health`, `/ready`, `/version`, `/metrics`, and `/debug/pprof*` bypass the envelope.

```json
{"status":"SUCCESS","response_code":"CORE-0000","message":"success","request_id":"...","timestamp":"...","data":{"item":"value"}}
```

**Excerpt, inside a gRPC handler:** set the registered application response code in the gRPC context; grpc-gateway carries it back as metadata.

```go
if err := response.SetCode(ctx, "CORE-0000"); err != nil {
	return nil, err
}
```

Use `gateway.Raw` or `gateway.RawHTTP` only when a route must return its own response contract. Normal API routes are enveloped by default. See [Error Handling](docs/ERROR_HANDLING.md).

### Response Code Registry

| | Guidance |
|---|---|
| What | A process-wide map from response codes to transport metadata. |
| When to use | Register service-specific codes before serving requests. |
| How to turn on | Built-ins register automatically; add custom codes with `errors.Register`. |
| How to turn off | Built-in codes cannot be disabled; do not register unused custom codes. |

The registry maps a stable code to a class, HTTP status, gRPC status, finality, retryability, and safe default message. This keeps transport mapping consistent across services.

| Code | Class | HTTP | gRPC | Finality | Retryable |
|---|---|---:|---|---|---|
| `CORE-0000` | `SUCCESS` | 200 | `OK` | Business | No |
| `CORE-1000` | `FAILED` | 400 | `InvalidArgument` | Business | No |
| `CORE-1001` | `FAILED` | 401 | `Unauthenticated` | Business | No |
| `CORE-1002` | `FAILED` | 401 | `Unauthenticated` | Business | No |
| `CORE-1003` | `FAILED` | 403 | `PermissionDenied` | Business | No |
| `CORE-1004` | `FAILED` | 404 | `NotFound` | Business | No |
| `CORE-1005` | `FAILED` | 413 | `ResourceExhausted` | Business | No |
| `CORE-9000` | `ERROR` | 500 | `Internal` | Technical Non-Recoverable | No |
| `CORE-9001` | `ERROR` | 503 | `Unavailable` | Technical Recoverable | Yes |

Classes also include `PENDING` and `UNKNOWN` for custom definitions. Register application codes at startup, before serving requests.

**Excerpt:**

```go
err := errors.Register(errors.Def{
	Code: "ORDER-1001", Class: errors.ClassFailed,
	HTTPStatus: 409, GRPCCode: codes.AlreadyExists,
	Finality: errors.FinalityBusiness, DefaultMessage: "order already exists",
})
```

`errors.Register` rejects duplicate or invalid definitions. An unregistered code resolves to `CORE-9000`; do not expose its internal message. Implement `errors.MessageProvider` and call `errors.SetMessageProvider` when a service needs localized/user-specific messages. See [errors.md](docs/errors.md).

### Raw Gateway Responses

| | Guidance |
|---|---|
| What | A per-handler opt-out from the success envelope. |
| When to use | Transparent proxies, webhooks, or file downloads with their own response format. |
| How to turn on | Wrap with `gateway.Raw` or `gateway.RawHTTP`. |
| How to turn off | Remove the wrapper; the normal envelope applies. |

`gateway.Raw` and `gateway.RawHTTP` opt one explicitly registered handler out of the 2xx envelope. Use them for a transparent proxy, webhook receiver, or file download when the upstream/body format must remain yours. Do not use them for ordinary APIs that should use the shared envelope. The handler owns status, headers, content type, body, and error format.

**Excerpt, `runtime.ServeMux.HandlePath`:**

```go
err := mux.HandlePath(http.MethodGet, "/partner/{id}", gateway.Raw(func(w http.ResponseWriter, r *http.Request, params map[string]string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"partner":"response"}`))
}))
```

For an `http.Handler`, wrap it with `gateway.RawHTTP(handler)` inside the `HandlePath` adapter. To find every registration in a service, run `rg -n 'gateway\.Raw(HTTP)?\('`. Operational paths remain exempt independently of Raw.

**Excerpt, adapting a plain `http.Handler`:**

```go
err := mux.HandlePath(http.MethodGet, "/download", func(w http.ResponseWriter, r *http.Request, _ map[string]string) {
	gateway.RawHTTP(downloadHandler).ServeHTTP(w, r)
})
```

### Request ID And Identity

| | Guidance |
|---|---|
| What | Request-scoped ID and identity values used in logs and propagation. |
| When to use | Add trusted user IDs or registered labels to request context. |
| How to turn on | Gateway/gRPC requests get a field bag automatically; set values with `SetUserID` and `SetLabel`. |
| How to turn off | Omit identity values and do not opt outbound HTTP clients into propagation. |

A request ID lets operators connect gateway logs, gRPC logs, and downstream calls. The gateway accepts a valid `X-Request-ID` or creates one, returns it in the response header, and stores it in the request field bag. `observability.SetUserID`, `SetLabel`, and `RegisterPropagatedLabel` set identity values in code.

**Excerpt, for a worker or background context:**

```go
ctx := observability.EnsureFieldBag(ctx)
_ = observability.SetUserID(ctx, userID)
_ = observability.SetLabel(ctx, "client_app", "app-a")
```

**Excerpt, at startup (once):**

```go
if err := observability.RegisterPropagatedLabel("client_app"); err != nil {
	return err
}
```

Register propagated labels once during startup; do not call `RegisterPropagatedLabel` per request.

`httpclient.WithInternalPropagation()` forwards request ID, user ID, and registered labels to HTTP destinations. It does not validate destination hosts. GC-27 records the trust boundary: HTTP propagation is caller-controlled, and Kafka identity headers are trusted based on internal network/producer controls, not per-message authentication. Forward identity only to destinations your service is authorized to trust.

### Logging

| | Guidance |
|---|---|
| What | ECS-compatible structured service, database, event, transaction, and access logs. |
| When to use | Record operational events and errors with the shared logger. |
| How to turn on | `app.New` creates the logger; set verbosity with `LOG_LEVEL`. |
| How to turn off | No global off switch; raise the log level or leave optional request-start logging disabled. |

Logs use ECS-compatible JSON, except `APP_ENV=local`, which uses a console encoder. The `app.log_type` categories include `access`, `service`, `db`, `event`, and `transaction`. Records also carry service identity (`service.name`, `service.environment`, `service.version`); when present, request context adds `labels.request_id`, `labels.gtid`, `user.id`, `client.ip`, `user_agent.original`, and trace/span IDs.

Access logs include route/method/status/response code/duration. They do not include request/response bodies or arbitrary headers.

Access class `ERROR` logs at error level; `UNKNOWN`, unregistered-code, replaced-request-ID, or slow events are warnings; ordinary access events are info. A slow request is marked when it exceeds `Log.SlowRequestThreshold`; slow database calls use `Log.SlowQueryThreshold`. Sensitive keys such as password, token, authorization, card, and account are redacted. Add service-specific sensitive keys via `Log.MaskKeys`.

Create the logger once during startup and pass its interface to infrastructure constructors. Logging is always available; `LOG_LEVEL` controls the minimum level, and `APP_ENV=local` selects the console encoder.

**Excerpt:**

```go
log, err := logger.New(cfg.App.ServiceName, cfg.App.LogLevel,
	logger.WithEnvironment(cfg.App.Environment),
	logger.WithConfig(logger.Config{MaskKeys: []string{"partner_secret"}}),
)
```

Use `Log.MaskKeys` for additional sensitive field names. The gRPC interceptor logs a request that reaches gRPC once at that boundary. Avoid logging the same error again at every layer. Successful direct `runtime.ServeMux.HandlePath` handlers that do not call gRPC currently have no access log; this is a known gap tracked in [the TODO backlog](docs/TODO_BACKLOG.md).

### Errors

| | Guidance |
|---|---|
| What | `AppError` carries a code, safe message, details, and optional cause. |
| When to use | Return application failures that need stable HTTP/gRPC mapping. |
| How to turn on | Create errors with `errors.New`, `Wrap`, or `Validation`; register custom codes at startup. |
| How to turn off | Error mapping is part of the transport contract; return success when no error occurred. |

`errors.AppError` carries a response code, safe message, optional validation details, and wrapped cause. `errors.New`, `errors.Wrap`, and `errors.Validation` create errors; the gRPC interceptor maps registered errors to gRPC status/details. Unknown internal errors are sanitized as `CORE-9000`.

**Excerpt:**

```go
return nil, errors.Validation("invalid input", errors.Detail{Field: "name", Reason: "required"})
```

### HTTP Client

| | Guidance |
|---|---|
| What | HTTP wrapper with timeout, logging, tracing, and a circuit breaker. |
| When to use | Outbound HTTP calls that need shared transport error classification. |
| How to turn on | Construct a client with `httpclient.NewClient` and per-client options. |
| How to turn off | Do not use the wrapper for that call; retries/body logging remain opt-in. |

The shared HTTP client adds structured logs, tracing, timeouts, and a circuit breaker. Retries are off by default; the breaker is on by default. A received HTTP response, including 5xx, is returned with a nil transport error. Connection and timeout failures return `*httpclient.TransportError` with `Phase` (`before_send` or `after_send`) and `Kind`.

**Excerpt:**

```go
client := httpclient.NewClient(log,
	httpclient.WithTimeout(5*time.Second),
	httpclient.WithRetry(resilience.DefaultRetryOptions()),
	httpclient.WithCircuitBreaker(resilience.DefaultCircuitBreakerOptions("partner")),
)
request := client.Request()
response, err := client.Do(ctx, request, http.MethodGet, "https://payments.internal/health")
```

Options apply to that client/endpoint; retries should be enabled only for safe operations. `WithBodyLogging` requires a masker. Identity propagation is a separate explicit option.

The breaker default trips after more than five consecutive failures and waits 60 seconds before half-open. See [HTTP client docs](httpclient/README.md).

### gRPC Client

| | Guidance |
|---|---|
| What | Outbound gRPC connection with call logs, tracing, identity, and deadlines. |
| When to use | Calls from a service to another gRPC service. |
| How to turn on | Build with `grpcclient.NewClient`; supply TLS credentials when required. |
| How to turn off | Do not create/use the client; metrics are off unless explicitly enabled. |

The gRPC client adds call logging, OpenTelemetry stats, identity propagation, keepalive, and a default 30-second deadline only when the call context has no deadline. Service-config retries are disabled. TLS credentials must be provided explicitly; plaintext is the default.

**Excerpt:**

```go
client, err := grpcclient.NewClient("payments.internal:50051", log,
	grpcclient.WithServiceName("catalog-api"),
	grpcclient.WithTransportCredentials(credentials.NewTLS(&tls.Config{})),
	grpcclient.WithMetricsEnabled(true),
)
```

### Kafka

| | Guidance |
|---|---|
| What | Kafka publisher and consumer helpers with SASL/TLS support. |
| When to use | Async events or partition-ordered message processing. |
| How to turn on | Set `KAFKA_BROKERS`, then create a publisher or consumer in code. |
| How to turn off | Leave `KAFKA_BROKERS` unset and do not create/start clients. |

SASL supports `PLAIN`, `SCRAM-SHA-256`, and `SCRAM-SHA-512`. `KAFKA_JKS_FILE` accepts JKS or PEM trust material; the matching credentials/TLS configuration is used by publishers, consumers, and readiness checks.

Consumer messages are processed in order within each partition. A failed offset blocks later offsets in that partition unless a successful DLQ publish permits commit. DLQ records keep original headers and add `x-dlq-reason` and `x-original-topic`. Retry and DLQ policies are explicit consumer options.

**Excerpt:**

```go
consumer, err := application.NewKafkaConsumer(
	"orders", "catalog-api",
	handleOrder,
	messaging.WithConsumerConcurrency(4),
	messaging.WithConsumerDLQ(dlqPublisher),
)
if err != nil {
	return err
}
go func() {
	if err := consumer.Start(ctx); err != nil && ctx.Err() == nil {
		log.Error(ctx, "Kafka consumer stopped", logger.Field{Key: "error", Value: err.Error()})
	}
}()
```

Kafka clients are not started merely by setting broker configuration: create the publisher/consumer in code and call `Start`. The consumer is closed by the app lifecycle. Omitting a DLQ/retry option leaves that behavior off.

### Outbox And Audit Events

| | Guidance |
|---|---|
| What | A transactional table plus a worker that publishes committed events. |
| When to use | A database update must not silently lose its corresponding event. |
| How to turn on | Call `PublishTx` inside the transaction and explicitly start a configured worker. |
| How to turn off | Do not publish outbox rows or create/start a worker. |

The outbox stores an event in the same SQL transaction as the business change. A service-owned worker publishes committed rows later, so a process crash does not silently lose the event. Rows are ordered by key: an older non-`SENT` row blocks later rows for that key. After the configured maximum attempts, the row becomes `DEAD` and continues to block that key until an operator resolves it.

For generic messages, put the consumer's idempotency key in your payload or headers; the SQL `event_id` is stored by the outbox but is not automatically exposed as a standard Kafka header. For audit events, consumers can use CloudEvents `ce_id` and persist deduplication with their side effect.

**Excerpt, inside a transaction:**

```go
publisher := outbox.NewPublisher(outbox.WithPublisherDriver("mysql"))
if err := publisher.PublishTx(ctx, tx, messaging.Message{Topic: "orders", Key: key, Payload: payload}); err != nil {
	return err
}
```

`PublishAudit` writes CloudEvents 1.0 headers: `ce_specversion`, `ce_id`, `ce_type`, `ce_source`, `ce_time`, `ce_subject`, and `ce_sequence`. It requires a UUID `ce_id`; `ce_subject` is the Kafka key.

Audit logging includes metadata, not event payload. See [Messaging Pattern](docs/MESSAGING_PATTERN.md).

**Excerpt, add an audit event to the caller's transaction:**

```go
err := publisher.PublishAudit(ctx, tx, outbox.AuditEvent{
	Topic: "audit",
	SpecVersion: "1.0",
	ID: "01234567-89ab-4cde-8fab-0123456789ab",
	Type: "com.example.order.created.v1",
	Source: "/catalog",
	Time: time.Now().UTC(),
	Subject: "order-123",
	Sequence: "1",
	Data: payload,
})
```

Stop the worker as part of service shutdown. Omitting worker creation does not delete rows from the outbox.

**Excerpt, when the database and Kafka publisher are enabled:**

```go
worker := outbox.NewWorkerWithOptions(
	application.SQLByName("primary").DB,
	eventPublisher,
	application.Logger(),
	outbox.WithWorkerDriver("mysql"),
)
if err := worker.StartChecked(ctx); err != nil {
	return err
}
defer worker.Stop(context.Background())
```

### Database And Migrations

| | Guidance |
|---|---|
| What | SQL Server/MySQL pools, GORM, transaction helpers, and Goose migration support. |
| When to use | Persistence or schema migration is part of the service. |
| How to turn on | Add aliases to `DB_LIST` or build `Databases` in code; run migrations under service policy. |
| How to turn off | Leave `DB_LIST` empty and migration autorun false. |

Supported SQL drivers are SQL Server and MySQL. `DB_LIST` selects one or more aliases; each becomes a `DB_<ALIAS>_*` group. Aliases are normalized to lowercase. Default pool limits are 20 open, 10 idle, 2-minute idle lifetime, and 5-minute connection lifetime; set alternatives in `cfg.Databases["primary"]`.

Migrations use Goose and are not automatic by default. `migration.AutoRunUp(cfg)` does nothing unless `cfg.Migration.AutoRun` is true and a DB alias/directory are configured. A service chooses when to run migrations.

**Excerpt, before `app.New`:**

```go
cfg.Databases["primary"].MaxOpenConns = 40
cfg.Migration.AutoRun = true
cfg.Migration.DBName = "primary"
cfg.Migration.Dir = "./migrations"
```

After loading and overriding Config, `app.New` opens each configured alias. Access a database with `application.SQLByName("primary")`.

**Excerpt, after `app.New` with the alias configured:**

```go
primary := application.SQLByName("primary")
sqlDB := primary.DB
gormDB := primary.Gorm()
```

`app.New` opens configured database aliases and applies their pool fields.

### Cache

| | Guidance |
|---|---|
| What | Shared `cache.Cache` operations backed by Redis or Memcached. |
| When to use | Values should be cached outside the service process. |
| How to turn on | Set `REDIS_ADDRESS` or `MEMCACHED_SERVERS`; use the app cache. |
| How to turn off | Leave both enabling values unset. |

`REDIS_PASSWORD` is optional. Enabled caches are required readiness dependencies.

**Excerpt, after `app.New` with the selected cache enabled:**

```go
value, err := application.RedisCache().Get(ctx, "catalog:42")
```

Both backends implement `cache.Cache` (`Get`, `Set`, `Delete`, `Health`, `Close`). `app.New` connects and checks enabled backends. The returned cache is nil when its integration is not configured.

### Tracing And Metrics

| | Guidance |
|---|---|
| What | OpenTelemetry OTLP traces and Prometheus runtime metrics. |
| When to use | Export traces to an OTLP gRPC collector or expose service metrics. |
| How to turn on | Set `APM_ENDPOINT`; set `METRICS_ENABLED=true` for metrics. |
| How to turn off | Leave `APM_ENDPOINT` empty and `METRICS_ENABLED=false`. |

`APM_ENDPOINT` enables OpenTelemetry OTLP over gRPC. Transport is secure by default; TLS/CA behavior is set through Config fields. The collector/APM server must accept OTLP gRPC.

`METRICS_ENABLED` defaults to `false`. When off, metric recording is gated and `/metrics` returns 404. Set it to `true` to record metrics and expose the scrape endpoint. The removed `app_transaction_total` metric is not available.

For code-controlled deployments, set fields after loading environment configuration:

**Excerpt:**

```go
cfg.Observability.OTLPEndpoint = "collector.internal:4317"
cfg.Metrics.Enabled = true
```


### Health, Readiness, And Shutdown

| | Guidance |
|---|---|
| What | Probe endpoints, gRPC health state, and graceful application shutdown. |
| When to use | Long-running services managed by probes or deployment termination. |
| How to turn on | Gateway/gRPC server register health support; run them with `server.Run`. |
| How to turn off | These behaviors are part of a constructed server; omit a transport only if the service does not expose it. |

`/health` reports that the process can answer. `/ready` reports whether required dependencies are available. On shutdown, readiness becomes 503 and gRPC health becomes `NOT_SERVING`; then the default drain delay is 3 seconds and the graceful shutdown budget is 25 seconds.

Set `App.DrainDelay` and `App.ShutdownTimeout` in code to fit the deployment termination grace period. `server.Run` coordinates the app and server components.

**Check locally:**

```sh
curl -i http://localhost:8080/health
curl -i http://localhost:8080/ready
```

These endpoints are mounted by the gateway. Readiness includes only enabled/required dependencies; shutdown transitions it to 503 automatically.

### Diagnostic Snapshot

| | Guidance |
|---|---|
| What | A bounded warning log with runtime memory, GC, heap, and goroutine data. |
| When to use | Diagnose a process that crosses a configured heap or goroutine threshold. |
| How to turn on | Enabled by default; thresholds and interval are Config fields. |
| How to turn off | Set `DIAG_ENABLED=false` or `cfg.Diagnostics.Enabled=false`. |

The watcher checks every 30 seconds and emits a warning snapshot when goroutines reach 5,000 or sampled heap-object bytes reach 80% of the cgroup memory limit. It logs reason, heap/memory numbers, goroutine count, GC counters, top 10 heap functions, and grouped top goroutine frames (up to 5 frames each).

A 10-minute cooldown limits repeated snapshots. If no cgroup memory limit is available, the heap trigger is disabled; the goroutine trigger still works.

No diagnostic snapshot is taken unless a configured threshold is crossed.

### Security

| | Guidance |
|---|---|
| What | HTTP signature/JWT checks, body limits, CORS, and optional pprof. |
| When to use | Validate trusted callers and protect public HTTP exposure. |
| How to turn on | Configure signature/JWT fields, body limit/CORS; enable pprof only outside production. |
| How to turn off | Keep signature/JWT disabled, CORS origins empty, and pprof false; body limit remains required. |

- **HTTP signatures:** a nonempty `AUTH_SIGNATURE_MASTER_KEY` enables HMAC-SHA256 validation of method, path, timestamp, and body. Defaults are `x-signature`, `x-timestamp`, and five-minute drift. The integration-specific signature contract still needs consumer confirmation; see `Q-SIG` in [Security](docs/SECURITY.md).
- **Internal JWT:** disabled by default. When enabled, choose exactly one key source: `PublicKey` for a static key rotated through deployment, or `JWKSEndpoint` for automatic rotation. Setting both or neither is rejected at startup. JWKS must use HTTPS except in `APP_ENV=local`. The verifier accepts RS256/384/512. Include-method and exclude-method lists cannot both be set.
- **Body limit:** default 1 MiB. Oversize requests return 413 with `CORE-1005`.
- **CORS:** disabled unless exact origins are configured. Do not use wildcard policy as a substitute for an origin allowlist.
- **pprof:** disabled by default. Enabling it in `prod`/`production` makes gateway construction fail.
- **Raw handlers:** bypass only the success envelope. The handler owns its entire response/error contract; other gateway security middleware still runs.

**Excerpt, code-controlled signature enablement:**

```go
cfg.Auth.Signature.MasterKey = signatureSecret
cfg.Auth.Signature.Enabled = true
```

Keep the key in a secret manager/environment injection path; do not commit it. Body-size limits, CORS, and pprof are controlled by `HTTP.MaxBodyBytes`, `HTTP.CORSAllowedOrigins`, and `HTTP.PprofEnabled`.

**Excerpt, code-controlled internal JWT enablement:**

```go
cfg.Auth.InternalJWT.Enabled = true
cfg.Auth.InternalJWT.PublicKey = publicKeyPEM
cfg.Auth.InternalJWT.IncludeMethods = []string{"/example.v1.Service/Update"}
```

### Testing Helpers

| | Guidance |
|---|---|
| What | Test-only recording logger, fake publisher, isolated registry, and HTTP fixtures. |
| When to use | Unit/integration tests need deterministic infrastructure substitutes. |
| How to turn on | Import `github.com/yogayulanda/go-core/coretest` from test code. |
| How to turn off | Do not import it from production packages; it has no runtime activation. |

`coretest` provides a recording logger, fake publisher, isolated response registry, and reusable HTTP fixtures. Use these to test service behavior without external Kafka/HTTP infrastructure.

**Excerpt:**

```go
coretest.WithIsolatedRegistry(t)
publisher := coretest.NewFakePublisher()
log, err := coretest.NewRecordingLogger()
```

These helpers are test-only; they do not change runtime behavior. Use `WithIsolatedRegistry(t)` when a test registers custom response codes so global registry state is restored at cleanup.

## Configuration Reference

Rule: environment variables configure infrastructure, credentials, and explicit kill switches. Non-empty invalid integer/boolean values fail during `config.Load()`; booleans must use a value accepted by `strconv.ParseBool` (`1`, `t/T`, `true/True/TRUE`, `0`, `f/F`, or `false/False/FALSE`). Set other runtime options as `Config` fields after `config.Load()` and before `Validate()`/`app.New()`. Setting an optional integration address/list enables it.

The full source-derived environment and code-field tables are in [Configuration Reference](docs/CONFIGURATION.md). That file also lists removed environment names, the local `.env` example, and code overrides. At startup, look for the `effective_config` log record; secret values are redacted.

## Operational Endpoints

| Endpoint | Behavior |
|---|---|
| `GET /health` | `200` with `ok` while the process responds. |
| `GET /ready` | `200` with dependency checks when ready; `503` when draining or a required dependency is down. |
| `GET /version` | Build/version metadata. |
| `GET /metrics` | Prometheus scrape when enabled; `404` when `METRICS_ENABLED=false`. |
| `/debug/pprof/*` | Available only when `HTTP.PprofEnabled=true`; startup refuses it in production. |
| gRPC health service | `SERVING` normally, `NOT_SERVING` during drain. |

## Common Mistakes / FAQ

1. **My JSON is not wrapped.** `/health`, `/ready`, `/version`, `/metrics`, and `/debug/pprof*` bypass the envelope. A handler wrapped with `gateway.Raw`/`RawHTTP` also opts out.

2. **My custom code becomes `CORE-9000`.** Register a valid `errors.Def` before serving requests. Duplicate/invalid registrations fail.

3. **My request ID is absent downstream.** The gateway and gRPC interceptor create the field bag for each incoming request. Use `observability.EnsureFieldBag` only for a context created outside a request, such as a worker or background goroutine. HTTP forwarding additionally needs `httpclient.WithInternalPropagation()`.

4. **That header did not propagate.** Propagation is opt-in and caller-controlled. It does not validate destination hosts; only enable it for trusted destinations.

5. **pprof refuses to start.** It is disabled by default and forbidden in `prod`/`production`.

6. **`/metrics` returns 404.** This is expected until `METRICS_ENABLED=true`.

7. **Kafka TLS is not enabled.** Configure `KAFKA_JKS_FILE` with JKS or PEM trust material; JKS requires its password. TLS configuration is shared across Kafka clients.

8. **An HTTP request does not retry.** Retries are off by default. Add `httpclient.WithRetry` only for safe operations.

9. **A circuit breaker opens unexpectedly.** The HTTP client breaker defaults to more than five consecutive failures and a 60-second open interval; configure it per client with `WithCircuitBreaker`.

10. **Migrations did not run.** `Migration.AutoRun` defaults false. Explicitly set migration Config fields or invoke migration code under service policy.

11. **A plain `HandlePath` route has no access log.** Successful direct handlers that do not call gRPC currently have this known gap. gRPC-backed routes log at the interceptor; Raw routes log at the envelope boundary. See [TODO backlog](docs/TODO_BACKLOG.md).

12. **A `DEAD` outbox event blocks later events.** Ordering is per key. Resolve or replay the older event before later rows for that key can proceed.

13. **Configuration fails at startup.** `config.Load` reports non-empty invalid integer and boolean values together; booleans accept only `1`, `t/T`, `true/True/TRUE`, `0`, `f/F`, or `false/False/FALSE`. Replace non-standard values such as `yes`, `on`, and `off`.

## Development

```sh
go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.14.0
make test
make vet
make lint
go test -race ./...
go build ./...
govulncheck ./...
```

`make check` runs the fast test/vet/lint gate. Release-oriented targets are in the Makefile. For performance regression decisions, use the locked-procs, interleaved benchmark method in [Technical Guide: Hot-Path Benchmark Method](docs/TECHNICAL_GUIDE.md#hot-path-benchmark-method); do not make a release decision from sequential WSL samples.

## Versioning And Go Policy

- The module path is `github.com/yogayulanda/go-core`; releases follow semantic versioning.
- GC-40 uses the latest patch in the N-1 Go minor line, advances with the February/August minor releases, and applies security patches within 14 days.
- The `go` directive is the minimum required by dependencies, not the installed patch. See [Migration](MIGRATION.md), [Versioning](docs/VERSIONING.md), and [Changelog](CHANGELOG.md).

## Further Docs

| Document | What it covers |
|---|---|
| [ARCHITECTURE](docs/ARCHITECTURE.md) | Package boundaries and runtime shape. |
| [CHANGE_CHECKLIST](docs/CHANGE_CHECKLIST.md) | Checklist for cross-package changes. |
| [CODING_STANDARD](docs/CODING_STANDARD.md) | Go style and implementation conventions. |
| [CONFIGURATION](docs/CONFIGURATION.md) | Complete environment and Config field reference. |
| [CONFIGURATION_PROFILES](docs/CONFIGURATION_PROFILES.md) | Configuration profiles and enablement. |
| [DATA_MODEL](docs/DATA_MODEL.md) | Persistence model guidance. |
| [DOMAIN](docs/DOMAIN.md) | Domain boundaries and ownership. |
| [ERROR_HANDLING](docs/ERROR_HANDLING.md) | Application errors and transport mapping. |
| [EVENT_CONTRACT](docs/EVENT_CONTRACT.md) | Event and CloudEvents contract. |
| [FOUNDATION_BOUNDARY](docs/FOUNDATION_BOUNDARY.md) | What belongs in the shared foundation. |
| [IDEMPOTENCY](docs/IDEMPOTENCY.md) | Idempotency patterns. |
| [MESSAGING_PATTERN](docs/MESSAGING_PATTERN.md) | Kafka, outbox, DLQ, ordering, and audit events. |
| [OBSERVABILITY](docs/OBSERVABILITY.md) | Logs, metrics, tracing, and diagnostics. |
| [PAGINATION](docs/PAGINATION.md) | Pagination contract and examples. |
| [PRODUCTION_SIGNOFF](docs/PRODUCTION_SIGNOFF.md) | Production readiness checks. |
| [QUICK_PROMPTS](docs/QUICK_PROMPTS.md) | Short contributor prompts. |
| [RELEASE_EVIDENCE_TEMPLATE](docs/RELEASE_EVIDENCE_TEMPLATE.md) | Release evidence format. |
| [RELEASING](docs/RELEASING.md) | Release process. |
| [RELIABILITY](docs/RELIABILITY.md) | Reliability practices and failure behavior. |
| [SECURITY_ADVISORIES](docs/SECURITY_ADVISORIES.md) | Dependency/toolchain vulnerability evidence. |
| [SECURITY](docs/SECURITY.md) | Security boundaries and consumer responsibilities. |
| [SERVICE_BOOTSTRAP](docs/SERVICE_BOOTSTRAP.md) | Service initialization guidance. |
| [TECHNICAL_GUIDE](docs/TECHNICAL_GUIDE.md) | Technical APIs, Raw gateway, and benchmark method. |
| [TODO_BACKLOG](docs/TODO_BACKLOG.md) | Deferred decisions and operational follow-ups. |
| [TRANSACTION_OBSERVABILITY](docs/TRANSACTION_OBSERVABILITY.md) | Transaction-oriented logs and observability. |
| [TRANSACTION_RULES](docs/TRANSACTION_RULES.md) | Transaction boundaries and rules. |
| [VERSIONING](docs/VERSIONING.md) | Compatibility and version policy. |
| [errors](docs/errors.md) | Error code model and examples. |
| [Benchmark evidence](docs/benchmarks/2026-10-14-go-1.26/README.md) | Interleaved Go 1.26 and GC-39 benchmark data. |

## Contributing, Security, License

See [CONTRIBUTING.md](CONTRIBUTING.md), [SECURITY.md](SECURITY.md), and [LICENSE](LICENSE).
