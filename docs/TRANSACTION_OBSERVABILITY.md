# Transaction Observability

`logger.TransactionLog` is an optional platform-standard contract for transaction-oriented services. It is distinct from SQL transaction orchestration in `dbtx`.

## Exported Fields

The current public struct contains:

- `Feature`
- `Product`
- `Status`
- `PreviousStatus`
- `ResponseCode`
- `Amount` (optional)
- `DurationMs`

Identity is resolved from the context field bag, including `labels.gtid` and `user.id`; there are no `TransactionID`, `UserID`, `Operation`, or `Metadata` fields on `TransactionLog`.

Logs are emitted with `app.log_type=transaction` and `app.transaction.*` fields. Do not place sensitive values in amount-adjacent metadata or other log fields.

## Metrics Status

`app_transaction_total{service,operation,status}` is currently registered but has no increment path. Whether to remove it or wire it to `LogTransaction` is unresolved in `GC-24`; consumers should not treat it as a live counter until that decision is made.
