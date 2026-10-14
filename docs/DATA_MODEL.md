# Data Model

`go-core` owns technical models only:

- `config.Config` and its runtime sections.
- `errors.AppError`, response-code definitions, and the gateway response envelope.
- `messaging.Message` and `messaging.CloudEventAttributes`.
- `messaging/outbox.Event` with identity `ID`, UUID `EventID`, topic/key/payload/headers, status, attempt, last error, lease/availability times, and sent time.

Business tables/entities stay in the service repository. Services create the outbox table through their own migrations using the SQL Server or MySQL DDL in `docs/MESSAGING_PATTERN.md`. Row `id` is a database identity for ordering; `event_id` is the idempotency/deduplication UUID.
