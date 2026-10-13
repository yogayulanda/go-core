# Messaging Pattern

## Transactional Outbox Schema

The outbox row ID is a database identity used for ordering; `event_id` is the UUID deduplication key. The worker leases eligible rows per message key, publishes outside the claim transaction, and records `SENT`, retry state, or `DEAD` independently per event. A `DEAD` row blocks later rows with the same key until operationally resolved.

SQL Server:

```sql
CREATE TABLE dbo.outbox_events (
	id BIGINT IDENTITY(1,1) NOT NULL PRIMARY KEY,
	event_id NVARCHAR(36) NOT NULL UNIQUE,
	topic NVARCHAR(255) NOT NULL,
	msg_key VARBINARY(512) NOT NULL,
	payload VARBINARY(MAX) NOT NULL,
	headers NVARCHAR(MAX) NOT NULL,
	status NVARCHAR(16) NOT NULL,
	attempt INT NOT NULL CONSTRAINT DF_outbox_attempt DEFAULT 0,
	last_error NVARCHAR(MAX) NULL,
	available_at DATETIME2(6) NOT NULL,
	locked_until DATETIME2(6) NULL,
	created_at DATETIME2(6) NOT NULL,
	sent_at DATETIME2(6) NULL
);
CREATE INDEX outbox_claim_idx ON dbo.outbox_events (status, available_at, locked_until, id);
CREATE INDEX outbox_key_order_idx ON dbo.outbox_events (msg_key, id, status);
```

MySQL:

```sql
CREATE TABLE outbox_events (
	id BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
	event_id VARCHAR(36) NOT NULL UNIQUE,
	topic VARCHAR(255) NOT NULL,
	msg_key VARBINARY(512) NOT NULL,
	payload LONGBLOB NOT NULL,
	headers JSON NOT NULL,
	status VARCHAR(16) NOT NULL,
	attempt INT NOT NULL DEFAULT 0,
	last_error TEXT NULL,
	available_at DATETIME(6) NOT NULL,
	locked_until DATETIME(6) NULL,
	created_at DATETIME(6) NOT NULL,
	sent_at DATETIME(6) NULL,
	INDEX outbox_claim_idx (status, available_at, locked_until, id),
	INDEX outbox_key_order_idx (msg_key, id, status)
);
```

Matching test fixtures are maintained at `messaging/outbox/testdata/schema_sqlserver.sql` and `messaging/outbox/testdata/schema_mysql.sql`.

Use this document as the canonical messaging and outbox consumption pattern for services using `go-core`.

Kafka authentication is configured with `KAFKA_USERNAME`, `KAFKA_PASSWORD`, and `KAFKA_SASL_MECHANISM` (`PLAIN`, `SCRAM-SHA-256`, or `SCRAM-SHA-512`). `KAFKA_JKS_FILE` accepts either a JKS truststore/keystore or a PEM bundle; `KAFKA_JKS_PASSWORD` is used for JKS files. Producer, consumer, and readiness checks share the same configuration. The office's selected SASL mechanism remains an operations question; all three mechanisms are supported.

## When To Use Direct Publisher

Use `messaging.Publisher` directly when:

- the action is not coupled to a SQL write
- losing the event on process failure is acceptable
- the service does not need outbox durability

Recommended ownership:

- service decides topic and key
- service decides retry and DLQ policy through publisher options
- publisher lifecycle stays explicit through `app.NewKafkaPublisher(...)`
- app-level helper now injects foundation logger + messaging metrics defaults automatically
- supports SASL PLAIN/SCRAM and JKS/PEM TLS via configuration
- propagates `x-request-id`, `x-user-id`, and registered propagated labels when headers are absent

## When To Use Outbox

Use `messaging/outbox` when:

- a SQL state change and event publication must succeed together logically
- the service needs durable asynchronous delivery
- retries and delayed publication are acceptable

Recommended pattern:

1. start SQL transaction with `dbtx.WithTx(...)`
2. write domain data through repository
3. write outbox record through `outbox.Publisher.PublishTx(...)`
4. commit once
5. let the outbox worker publish pending rows later

`PublishAudit` uses CloudEvents 1.0 headers (`ce_specversion`, `ce_id`, `ce_type`, `ce_source`, `ce_time`, `ce_subject`, `ce_sequence`), requires a UUID ID, and logs only event metadata. The consumer restores context identity and exposes validated/received CloudEvents fields through `messaging.CloudEventsFromContext(ctx)`.

## Worker Ownership

`go-core` intentionally does not start the outbox worker automatically.
The consuming service owns:

- when the worker starts
- what interval and batch size are used
- what publisher, logger, and metrics are attached
- whether the worker runs in the current process topology

This keeps runtime behavior explicit and service-controlled.

Recommended runtime hooks:

- use `outbox.Worker.StartChecked(ctx)` for service-owned background execution
- use `outbox.Worker.RunOnce(ctx)` for tests, admin jobs, or explicit single-batch control

Worker claims are leased and committed in a short database transaction. Kafka publish happens outside the transaction, then the row is independently marked `SENT`, returned to `PENDING` with an incremented attempt/backoff, or moved to `DEAD` at the attempt limit. Ordering is enforced per key; a `DEAD` row blocks later rows for that key. Consumers process and commit each partition in offset order; configured concurrency is across partitions. Successful DLQ publication preserves input headers and adds `x-dlq-reason` and `x-original-topic`.

## Runtime Observability

Publisher emits:

- `message_publish` via `logger.ServiceLog`
- `app_message_publish_total{service,topic,status}`

Consumer emits:

- `message_consume` via `logger.ServiceLog`
- `app_message_consume_total{service,topic,group,status}`
- `app_message_process_duration_seconds{service,topic,group}`

Outbox worker emits:

- `outbox_worker` via `logger.ServiceLog`
- `outbox_batch` via `logger.ServiceLog`
- `app_outbox_batch_total{service,status}`
- `app_outbox_batch_duration_seconds{service}`
- `app_outbox_batch_size{service}`

## Retry, DLQ, and Success Logging

Recommended baseline:

- enable publisher retry for transient delivery failures
- enable DLQ only when the service has an operational plan for replay
- enable success logging selectively to avoid noisy logs at scale

## See Also

- `examples/outbox_example.go`
- `examples/bootstrap_example.go`
- `docs/TRANSACTION_RULES.md`
