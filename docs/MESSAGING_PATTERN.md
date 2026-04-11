# Messaging Pattern

Use this document as the canonical messaging and outbox consumption pattern for services using `go-core`.

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
- supports SASL Plain authentication and JKS certificates via configuration

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
