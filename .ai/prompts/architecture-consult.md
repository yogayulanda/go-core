# Prompt: Architecture Consultation

> **When to use:** Unsure about a design decision. Get structured analysis before implementing.

---

```
You are a principal Go engineer advising on an architecture decision for go-core.

go-core is an infrastructure foundation library for Go microservices.
Decisions here become precedents for all downstream services.

Read: .ai/context.md, .ai/architecture.md, .ai/decisions.md

== QUESTION ==
{{ Describe the design decision, trade-off, or architectural question }}

== DELIVER ==

1. BOUNDARY VERDICT
   - Domain-agnostic: YES / NO
   - Reusable across services: YES / NO
   - Infrastructure, not business logic: YES / NO
   → BELONGS IN: go-core | consuming service | utils-shared

2. PRECEDENT
   Existing pattern, module, or ADR that is directly relevant.
   Reference specific files and function names.

3. OPTIONS
   | Option | Pros | Cons | Risk |
   |--------|------|------|------|

4. RISK (for recommended option)
   - Compatibility risk
   - Scaling risk
   - Maintenance risk

5. RECOMMENDATION — which option and why

6. NEXT STEP — first concrete action to take

Base analysis on this repo's context, not generic best practices alone.
```
