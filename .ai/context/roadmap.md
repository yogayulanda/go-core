Current Improvement Roadmap

Priority order:

1. strengthen bootstrap examples and starter templates
2. formalize messaging and outbox service patterns
3. raise observability contracts for service and DB flows
4. improve config ergonomics and structured validation
5. strengthen CI, release, and adoption workflow

Current decisions:

- `TransactionLog` remains in `go-core`
- `LogTransaction(...)` remains in `go-core`
- `app_transaction_total` remains in `go-core`
- `UserID` stays top-level in `TransactionLog`
- `dbtx` is separate technical infrastructure and should not be conflated with transaction observability
- `Validate()` stays as the compact public entry point
- structured config validation is additive through `ValidateIssues()`

See also:

- `.ai/context/backlog.md` for untouched-area priority after the current workstream
