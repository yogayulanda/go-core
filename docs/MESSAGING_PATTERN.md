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
- what publisher and logger are attached
- whether the worker runs in the current process topology

This keeps runtime behavior explicit and service-controlled.

## Retry, DLQ, and Success Logging

Recommended baseline:

- enable publisher retry for transient delivery failures
- enable DLQ only when the service has an operational plan for replay
- enable success logging selectively to avoid noisy logs at scale

## See Also

- `examples/outbox_example.go`
- `docs/TRANSACTION_RULES.md`
