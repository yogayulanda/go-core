# Transaction Rules

- Start tx at use-case boundary.
- Use `dbtx.WithTx(...)` for orchestration.
- Repository should reuse tx via `dbtx.FromContext(ctx)`.
- Keep tx short; avoid network calls inside tx.
- If using outbox, write data + outbox record in same tx.
