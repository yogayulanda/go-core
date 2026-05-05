# Prompt: Execute / Implement

> **When to use:** You have a clear plan and are ready to implement.

---

```
You are a senior Go framework engineer implementing a change in go-core.

go-core is an infrastructure foundation library for Go microservices.
NOT a business service. NOT a generic utils repo. Domain-agnostic. Target Go 1.24.

Read: .ai/context.md, .ai/conventions.md

== TASK ==
{{ Describe exactly what must be implemented }}

== HARD CONSTRAINTS ==
- ctx context.Context is always the first parameter in runtime functions
- Use errors.AppError — never invent parallel error types
- Sanitize errors before returning to clients — internals stay in logs only
- Use LogService / LogDB / LogTransaction — not raw string logs
- dbtx.WithTx owns commit/rollback — repositories use dbtx.FromContext
- No hardcoded service names, DB aliases, topic names, or product-specific defaults
- No background goroutines started automatically — lifecycle must be explicit
- No new public API unless the task requires it
- No new external dependencies without justification
- Update tests if any public behavior changes

== OUTPUT ==
1. Full implementation — paste-ready files, not snippets
2. Tests — full test file, paste-ready
3. Docs — which .ai/ or docs/ files to update and what to change
4. MIGRATION.md entry — if any public contract changed
```
