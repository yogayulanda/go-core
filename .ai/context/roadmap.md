Current Improvement Roadmap

Current stabilization follow-through:

1. keep CI, scripts, and release docs aligned
2. keep versioning and upgrade expectations explicit
3. keep `.ai` context synchronized with repo reality

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
