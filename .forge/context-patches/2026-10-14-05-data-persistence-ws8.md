# Proposed Context Patch: 05 Data And Persistence

Status: proposal only; curated context remains unchanged.

## Replace stale confirmed facts
- Supported runtime SQL drivers are SQL Server and MySQL only. PostgreSQL support is not part of this release state.
- Replace prior outbox model/status notes with identity `id`, unique UUID `event_id`, `msg_key`, payload/headers, status (`PENDING`, `PROCESSING`, `SENT`, `DEAD`), attempt/error/availability/lease timestamps, creation and sent timestamps.

## Add confirmed facts
- `PublishTx` writes the outbox row in the caller-owned transaction; worker claims are short transactions, Kafka publish is outside the transaction, and finalization is independent per row.
- Ordering is enforced per key. A `DEAD` row blocks later rows for the same key.
- SQL Server/MySQL schemas are in `docs/MESSAGING_PATTERN.md` and `messaging/outbox/testdata/`.
- SQL integration tests use the `integration` build tag and require disposable real DBs; they compiled but were not run because Docker was unavailable (S8).

Evidence: `database/sqlserver.go`, `messaging/outbox/model.go`, `messaging/outbox/repository.go`, `messaging/outbox/worker.go`, schema fixtures.
