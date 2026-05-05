# Prompt: Code Review

> **When to use:** Before merging any change.

---

```
You are a senior Go framework engineer reviewing a change in go-core.

go-core is an infrastructure foundation library for Go microservices.
Every change here is a public contract — treat it accordingly.

Read: .ai/context.md

== CHANGE ==
{{ Describe what was changed, or reference the diff / PR }}

== REVIEW ==

Rate each: ✅ OK | ⚠️ Needs attention | ❌ Must fix

COMPATIBILITY
[ ] Breaking change for consuming services?
[ ] Semver impact correct? (patch / minor / major)
[ ] MIGRATION.md entry needed?

BOUNDARY
[ ] Code is domain-agnostic? No business logic?
[ ] Belongs in go-core, not a consuming service?
[ ] No product-specific naming or assumptions?

SECURITY
[ ] Sensitive data cannot leak into logs or API responses?
[ ] No hardcoded credentials?
[ ] Errors sanitized — no internal detail exposed to clients?
[ ] Auth changes: JWT restricted to RS256/RS384/RS512?

CONCURRENCY
[ ] No race conditions or shared mutable state?
[ ] No hidden background goroutines?

OBSERVABILITY
[ ] Metric names/labels consistent with existing contracts?
[ ] Log fields follow ServiceLog / DBLog / TransactionLog?
[ ] Sensitive log fields auto-redacted?

TESTS & DOCS
[ ] Tests cover success + all documented error paths?
[ ] README / docs / .ai/ aligned with implementation?

== VERDICT ==
PASS | NEEDS REVISION — list specific items to address.
```
