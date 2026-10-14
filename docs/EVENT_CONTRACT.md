# Event Contract

`messaging.Message` transports a topic, key, payload, headers, and optional `EventID`. Generic event payload schemas remain owned by consuming services.

## Audit Events

`messaging/outbox.Publisher.PublishAudit(ctx, tx, event)` writes an audit event to the outbox in the caller's SQL transaction. It validates CloudEvents 1.0 metadata and uses the event subject as Kafka key and `ce_id` as the stable `event_id`.

Required attributes:

- `ce_specversion=1.0`
- `ce_id` (UUID)
- `ce_type` matching `^[a-z0-9]+(\.[a-z0-9_]+)+\.v[0-9]+$`
- `ce_source`
- `ce_time`
- `ce_subject`
- `ce_sequence`

These attributes are stored as Kafka binary headers; `data` is carried only as the message payload. A successful publish logs `ce_type`, `ce_id`, `ce_subject`, and `ce_sequence` plus context identity, never payload values.

Consumers restore request/user/registered labels and CloudEvents attributes. Read the latter with `messaging.CloudEventsFromContext(ctx)`. Messages without `x-request-id` receive a generated request ID.

Consumers must be idempotent. Generic inbox support (`GC-37`) is not included in this release.
