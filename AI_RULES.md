# AI Rules

> Lens utama untuk semua pekerjaan AI di repo ini.
> **Baca `.ai/context.md` dulu sebelum apapun.**

## Hard Constraints

- Keep `go-core` domain-agnostic — no business entities, no service-specific defaults.
- Allow explicit platform-standard technical contracts when intentionally standardized across services.
- Prefer additive changes; avoid breaking public API.
- Keep public API surface small.
- No hidden lifecycle or background behavior — all lifecycle hooks must be explicit.

## Implementation Rules

- Runtime functions: `ctx context.Context` always first.
- Reuse `errors.AppError` — never invent parallel error types.
- Sanitize external error responses — internal detail stays in logs only.
- Use `LogService`, `LogDB`, or `LogTransaction` — not raw string logs for structured flows.
- `dbtx.WithTx` owns commit/rollback — repositories use `dbtx.FromContext`.

## Review Checklist (always mention)

- **Compatibility** — does this break existing consuming services?
- **Coupling** — does this introduce product-specific knowledge?
- **Concurrency** — new goroutines or shared state?
- **Scale risk** — metric cardinality explosion or connection growth?
- **Overengineering** — simpler than the problem requires?

## Context Navigation

| Question | Read |
|---|---|
| What is this repo? | `.ai/context.md` |
| System design & layers | `.ai/architecture.md` |
| Module APIs & symbols | `.ai/modules.md` |
| Auth, JWT, secrets | `.ai/security.md` |
| Transactions, retry, outbox | `.ai/transactions.md` |
| Request lifecycle & metrics | `.ai/data-flow.md` |
| External services & env vars | `.ai/integrations.md` |
| Code style & naming | `.ai/conventions.md` |
| Why decisions were made | `.ai/decisions.md` |
| Dev workflow & checklist | `.ai/workflow.md` |

## Output Style

Short, direct, minimal tokens. No filler sentences.
