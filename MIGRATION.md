# Migration Guide

This guide covers migration from the public v1.0.x line to v1.1.0.

## Upgrading to v1.2.1

- A non-empty malformed integer or boolean environment value now fails during `config.Load` instead of silently using its default. Boolean variables accept exactly the values recognized by `strconv.ParseBool`: `1`, `t`, `T`, `TRUE`, `true`, `True`, `0`, `f`, `F`, `FALSE`, `false`, and `False`. Values such as `yes`, `on`, and `off` are rejected.
- Before upgrading, check deployment environment files and ConfigMaps for non-standard boolean spellings such as `yes`, `on`, and `off`, plus malformed integer values. For example, inspect manifests and env files with `grep -RniE '(^|[=:])[[:space:]]*(yes|no|on|off)([[:space:]]|$)' <deployment-config-paths>` and confirm integer settings contain decimal digits only.
- Enabled Internal JWT verification requires exactly one key source. Choose `PublicKey` for a static key rotated through deployment, or `JWKSEndpoint` for automatic key rotation. Setting both is rejected at startup. JWKS URLs must use HTTPS except in `APP_ENV=local`.
- HTTP and gRPC ports, and configured database alias ports, must be within 1–65535; HTTP and gRPC ports must differ.

## Public v1.0.x History

- `v1.0.0` established the public stable baseline.
- `v1.0.1` was an OSS-readiness patch; it intended no runtime or public Go API changes.

## v1.1.0 Changes

The following changes are consumer-visible. Plan them together with consuming-service migrations.

### Configuration and databases

- Only `sqlserver` and `mysql` database drivers are supported.
- Config loading now uses the supported env contract in `.env.example` and `docs/CONFIGURATION_PROFILES.md`. Removed per-DB pool/TLS/shutdown/migration env overrides do not change runtime settings; set those values in code configuration instead.
- `KAFKA_USERNAME` requires `KAFKA_PASSWORD` and `KAFKA_SASL_MECHANISM` (`PLAIN`, `SCRAM-SHA-256`, or `SCRAM-SHA-512`). `KAFKA_JKS_FILE` accepts JKS or PEM trust material; `KAFKA_JKS_PASSWORD` is for JKS.

### Errors and gateway responses

- Error codes are open strings resolved by the registry. Use `errors.New`, `errors.Wrap`, and `errors.Validation`; register custom definitions with `errors.Register`. The previous closed category/domain/build surface is not supported.
- Gateway responses use `status`, `response_code`, `message`, `request_id`, and `timestamp`; successful responses include `data`; validation failures may include `details`. `CORE-1005` represents HTTP 413 body-limit failures. Unknown codes resolve to `CORE-9000`.
- Response-code metadata is transported between gRPC and the gateway and stripped from public HTTP headers.

### Logging and request identity

- Logs use ECS JSON except `APP_ENV=local`, which uses the console encoder. Identity uses `labels.request_id`, `labels.gtid`, `user.id`, `client.ip`, and `user_agent.original`.
- Access/service/database/event/transaction fields use the documented `app.*` and ECS fields. Consumers with dashboard queries based on legacy keys must update them.
- A gateway request that reaches gRPC has one access log from the gRPC interceptor. Gateway logs are reserved for failures before gRPC.

### HTTP and gRPC clients

- HTTP retries are disabled by default; opt in per client with `httpclient.WithRetry`.
- Received HTTP responses, including 5xx, return `(response, nil)`. Transport failures return `*httpclient.TransportError` with phase and kind.
- Request ID, user ID, and registered labels are forwarded by HTTP only with `WithInternalPropagation()`; they are never forwarded by default. Body logging requires `WithBodyLogging(masker)` and has no default-on mode.
- `grpcclient.NewClient` adds request identity propagation, OTel stats, call logging, keepalive, and a 30-second deadline only when the context has no deadline. Metrics require `grpcclient.WithMetricsEnabled(true)`; service-config retries are disabled. Configure transport credentials explicitly for TLS; plaintext matches the default server configuration.

### Kafka, outbox, and audit events

- Kafka consumer concurrency is assigned by partition; each partition is processed and committed in order. Failed processing blocks later offsets on that partition unless the message is successfully sent to DLQ and committed.
- DLQ messages preserve original headers and add `x-dlq-reason` and `x-original-topic`.
- Existing outbox tables are incompatible. Migrate to identity `id`, UUID `event_id`, `topic`, `msg_key`, `payload`, `headers`, `status`, `attempt`, `last_error`, `available_at`, `locked_until`, `created_at`, and `sent_at` using the SQL Server or MySQL schema in `docs/MESSAGING_PATTERN.md`.
- Outbox workers claim rows with leases, publish outside the claim transaction, retry with backoff, and mark exhausted events `DEAD`. `DEAD` blocks later events for the same key.
- `outbox.Publisher.PublishAudit` validates CloudEvents 1.0 attributes including UUID `ce_id`, `ce_subject`, and `ce_sequence`; consumers read restored values with `messaging.CloudEventsFromContext(ctx)`.

### Diagnostics and shutdown

- The gateway uses matched route-template metric labels; unmatched requests use `unmatched`.
- `DIAG_ENABLED` defaults to true. The watcher emits bounded diagnostic snapshots on configured goroutine/cgroup-heap thresholds, with a shared cooldown. Disable with `DIAG_ENABLED=false` when required by local policy.
- Shutdown defaults are `DrainDelay=3s` and `ShutdownTimeout=25s`; readiness returns 503 and gRPC health becomes `NOT_SERVING` during drain.
- Prometheus metric recording and `/metrics` are disabled by default. Set `METRICS_ENABLED=true` to enable them. The `app_transaction_total` metric was removed.
- Tracing remains OpenTelemetry OTLP to Elastic APM Server through `APM_ENDPOINT`; there is no Elastic APM agent and no `transaction.id` log field.

### Constructor and test-helper changes

- `logger.New` accepts functional options (`logger.WithEnvironment`, `logger.WithConfig`, `logger.WithOutput`). `logger.NewWithOptions` is removed.
- `database.New` accepts `database.WithSlowQueryThreshold`. `database.NewWithLogConfig` is removed.
- `coretest` is a test-only helper package for a recording logger, fake publisher, response registry isolation, and reusable HTTP transport fixtures.

## v1.2.0 (Unreleased)

### Go toolchain requirement

- Consumers must build with Go **1.26.0 or newer**. This is the minimum declared by the selected dependency graph after `go mod tidy`; it is not the Go patch version used by CI. CI and release validation use Go 1.26.9.
- The project follows a rolling N-1 policy: use the newest patch in the N-1 line, move to the next minor after Go's February/August releases, and apply security patches within 14 days. Security-fix minimum versions from official maintainers may be used sooner when needed to remediate an advisory.

### Runtime and standard-library behavior to verify

| Change | Consumer impact | Assessment for go-core |
|---|---|---|
| Go-version-specific `GODEBUG` defaults follow the module's `go` version | Raising the directive opts into newer compatibility defaults; explicit `GODEBUG` settings can override some behaviors and mask validation. | No `GODEBUG` override or `godebug` directive is configured. Validate the defaults below without adding overrides. |
| Go 1.25 container-aware, periodically updated `GOMAXPROCS` | Worker and request parallelism can follow cgroup CPU limits and limit changes instead of startup logical CPU count. Explicit `GOMAXPROCS` disables the automatic behavior. | No override in go-core; load-test services using CPU limits and worker concurrency. |
| Go 1.26 Green Tea GC enabled by default | GC throughput and pause/CPU profile can differ; the expected improvement is workload-dependent. | Diagnostics include GC pause measurements; compare benchmarks and production-like GC telemetry. |
| Go 1.25 TLS rejects SHA-1 signature algorithms in TLS 1.2; Go 1.26 enables post-quantum hybrid key exchanges by default | Legacy TLS peers may fail handshakes; middleboxes or peers may have compatibility/performance implications. | go-core configures TLS 1.2 minimum for Kafka truststores and custom roots for OTLP; test actual brokers, collectors, and downstream TLS peers. No `GODEBUG` override is set. |
| Go 1.25 `crypto/x509` changes certificate SubjectKeyId generation and rejects malformed certificate encodings more consistently | Generated certificates can have a different SubjectKeyId; malformed legacy certificates previously accepted may be rejected. | Truststore parsing is in use; current tests pass, but validate deployed JKS/PEM certificates. |
| Go 1.26 `net/url.Parse` rejects malformed host colons | Invalid URLs such as `http://host:80:80/` that were previously accepted now fail parsing. | HTTP client, health checks, and JWT JWKS setup parse URLs; validate configured endpoints and reject malformed values. |
| Go 1.26 `net/http` changes ServeMux trailing-slash redirects from 301 to 307 and scopes cookies using `Request.Host` | Redirect method preservation and cookie selection may differ for standard-library mux/client users. | Production gateway uses grpc-gateway's `runtime.ServeMux`; no production `http.ServeMux` or cookie jar use was found. No direct impact identified. |
| Go 1.26 TLS/x509, random-source, and RSA API behavior changes | TLS certificate signer behavior is more explicit; crypto APIs ignore caller-supplied randomness in several key/sign operations, and legacy RSA PKCS#1 v1.5 encryption is deprecated. | No direct calls to the affected random-argument APIs or legacy RSA encryption were found; consumers with custom signers or deterministic crypto tests should review them. |

No `GODEBUG` compatibility override is configured by go-core. Run service-specific TLS, endpoint validation, cgroup load, and performance checks before rollout.

### Hot-path benchmark evidence

An interleaved A/B/C comparison was run on 2026-10-14 with `GOMAXPROCS=4`, 20 rounds per variant, and `-run='^$' -benchmem -count=1` for each invocation. Benchstat found no statistically significant latency difference for the gateway or unary interceptor chain:

Raw samples and complete benchstat outputs are archived under [`docs/benchmarks/2026-10-14-go-1.26/`](docs/benchmarks/2026-10-14-go-1.26/).

| Benchmark | A: main / Go 1.24.13 | B: main / Go 1.26.9 | C: branch / Go 1.26.9 | A to B | B to C |
|---|---:|---:|---:|---:|---:|
| Gateway handler via `httptest.NewRecorder` | 28.08 us/op | 27.15 us/op | 29.62 us/op | -3.3%, p=0.799 | +9.1%, p=0.398 |
| Unary interceptor chain | 40.74 us/op | 43.62 us/op | 46.19 us/op | +7.1%, p=0.383 | +5.9%, p=0.947 |

Gateway allocations fell from 126 to 116 allocs/op between A and B (7.94% fewer), with no change between B and C; this is attributed to the toolchain, not dependencies. The matrix did not show a significant latency regression. Measurements run on WSL are noisy, with practical detection power around 10%; sub-10% differences should not be treated as reliable regressions without a stable Linux runner.

The gateway benchmark is in-process: it uses `httptest.NewRecorder`, not a TCP listener, and does not traverse a grpc-gateway-to-gRPC hop. That proxy path remains unbenchmarked. The Go 1.26.9 build rejects `GOEXPERIMENT=nogreenteagc` (`unknown GOEXPERIMENT greenteagc`), so a no-Green-Tea-GC comparison was unavailable.

## Still Blocked / Deferred

- `Q-SIG`: compatibility confirmation is deferred but is not a release blocker.
- v1.2.0 raises the minimum Go toolchain requirement to 1.26.0; the validated patch is Go 1.26.9.
- Downstream forks may adjust the module path before adoption.
- SQL Server/MySQL outbox integration tests require disposable databases and remain unverified against real database instances.
