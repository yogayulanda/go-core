# AI Rules (Compact)

Use `.ai/go-core.md` lens first.

## Hard Constraints
- Keep `go-core` domain-agnostic.
- Prefer additive changes; avoid breakage.
- Keep public API small.
- No hidden lifecycle/background behavior.

## Implementation Rules
- Runtime functions: `ctx context.Context` first.
- Reuse `errors.AppError` contract.
- Sanitize external error responses.
- Keep internals in logs, not API payload.

## Review Checklist
Always mention briefly:
- compatibility,
- coupling,
- concurrency,
- scale risk,
- overengineering risk.

## Output Style
Short, direct, minimal tokens.
