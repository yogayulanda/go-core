# Transaction Observability

`go-core` provides a platform-standard transaction observability contract for transaction-oriented services.

## Contract Surface

- `logger.TransactionLog`
- `logger.Logger.LogTransaction(ctx, tx)`
- Prometheus metric `app_transaction_total{service,operation,status}`

## Stable Top-Level Fields

`TransactionLog` keeps these top-level fields intentionally small and stable:

- `Operation`
- `TransactionID`
- `UserID`
- `Status`
- `DurationMs`
- `ErrorCode`
- `Metadata`

## Field Guidance

- `Operation`:
  a stable transaction flow or step name such as `payment_process`, `transfer_submit`, `refund_finalize`
- `TransactionID`:
  the business transaction identifier used for monitoring and correlation
- `UserID`:
  the actor or end-user identifier when the transaction can be tied to a user
- `Status`:
  prefer stable values such as `success`, `failed`, `pending`
- `DurationMs`:
  transaction processing duration in milliseconds
- `ErrorCode`:
  optional stable classification for alerting and dashboard grouping
- `Metadata`:
  extension area for additional structured attributes that are not standardized top-level fields

## Notes

- `UserID` may be empty for system-to-system or fully internal flows.
- Do not keep growing top-level fields unless the new field is truly needed as a cross-service platform standard.
- Do not put sensitive values into `Metadata`.
- This contract is separate from `dbtx`, which is only for SQL transaction orchestration.
- For normal technical service flow, use `logger.ServiceLog`.
- For DB operational or query-related logging, use `logger.DBLog`.
