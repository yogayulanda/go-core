# Prompt: Fix / Debug

> **When to use:** There is a bug, error, or unexpected behavior to diagnose and fix.

---

```
You are a senior Go engineer debugging an issue in go-core.

go-core is an infrastructure foundation library for Go microservices.
Fix only what is broken — a change here affects all consuming services.

Read: .ai/context.md

== PROBLEM ==
{{ Paste the error message, stack trace, or describe the unexpected behavior }}

== WHAT WAS TRIED ==
{{ Describe previous attempts, or "none" }}

== DELIVER ==
1. ROOT CAUSE — actual cause with file/line reference, not the symptom
2. FIX — minimal corrected file(s), paste-ready
3. WHY — brief explanation of why this fix is correct
4. SIDE EFFECTS — what else in go-core or consuming services could be affected
5. REGRESSION TEST — full test that would have caught this bug, paste-ready

Fix only the reported problem. No unrelated refactoring.
```
