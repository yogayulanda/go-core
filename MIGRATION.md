# Migration Guide

This file distinguishes the public tagged v1 line from the prepared internal foundation refresh. The internal refresh described below is **not released or tagged**; `D-PATH` remains open. Apply it only when the internal module path and adoption window are approved.

## Public v1.0.x History

- `v1.0.0` established the public stable baseline.
- `v1.0.1` was an OSS-readiness patch; it intended no runtime or public Go API changes.

## Prepared Internal 1.0.0 Refresh

The following changes are consumer-visible breaking changes. Plan them together with service migrations; do not infer that a released internal module/tag exists.

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
- `grpcclient.NewClient` adds request identity propagation, OTel stats, per-call logging/metrics, keepalive, and a 30-second deadline only when the context has no deadline. Service-config retries are disabled. Configure transport credentials explicitly for TLS; plaintext matches the default server configuration.

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

### Constructor and test-helper changes

- `logger.New` accepts functional options (`logger.WithEnvironment`, `logger.WithConfig`, `logger.WithOutput`). `logger.NewWithOptions` is removed.
- `database.New` accepts `database.WithSlowQueryThreshold`. `database.NewWithLogConfig` is removed.
- `coretest` is a test-only helper package for a recording logger, fake publisher, response registry isolation, and reusable HTTP transport fixtures.

## Still Blocked / Deferred

- `GC-23`: Elastic tracing backend/agent policy.
- `GC-24`: metrics enablement policy.
- `D-PATH`: internal module path; no tag was created.
- `Q-SIG`: confirmation of the existing HTTP signature scheme.
- SQL Server/MySQL outbox integration tests require Docker and were not run in the implementation environment (S8).
