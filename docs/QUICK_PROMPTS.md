# QUICK_PROMPTS

## Cara Pakai
1. Pilih template sesuai kebutuhan.
2. Isi placeholder `<...>` saja.
3. Kirim ke AI tanpa ulang konteks panjang.
4. Jika perlu, tambahkan 1-2 constraint ekstra.

## Global Prefix (pakai di semua template)
```text
Use CONTEXT.md + AI_RULES.md + docs/ARCHITECTURE.md + docs/CODING_STANDARD.md.
Keep response token-efficient.
```

## 1) Implement Feature
```text
Use CONTEXT.md + AI_RULES.md + docs/ERROR_HANDLING.md + docs/TRANSACTION_RULES.md.
Task: <fitur yang ingin dibuat>.
Scope: <file/packagenya>.
Constraints: domain-agnostic, additive-only, no breaking change, no hidden goroutine.
Output format: short plan -> implement patch -> test command -> risk note.
```

## 2) Code Review (Bug/Risk First)
```text
Use CONTEXT.md + AI_RULES.md + docs/CODING_STANDARD.md.
Review target: <PR/file/commit>.
Focus: bug, behavioral regression, reliability risk, missing tests.
Output format: findings by severity with file:line, then assumptions/open questions.
```

## 3) Debug Issue
```text
Use CONTEXT.md + AI_RULES.md + docs/RELIABILITY.md + docs/OBSERVABILITY.md.
Issue: <error/gejala>.
Context: <kapan terjadi + log penting>.
Goal: find root cause and propose minimal safe fix.
Output format: hypothesis -> verification steps -> fix patch -> post-fix checks.
```

## 4) Safe Refactor
```text
Use CONTEXT.md + AI_RULES.md + docs/CODING_STANDARD.md.
Refactor target: <fungsi/package>.
Goal: improve readability/maintainability without behavior change.
Constraints: public API stable, additive changes only.
Output format: plan -> patch -> equivalence notes -> tests run.
```

## 5) Add/Improve Tests
```text
Use CONTEXT.md + AI_RULES.md + docs/CODING_STANDARD.md + docs/RELIABILITY.md.
Test target: <package/function>.
Focus: edge cases, failure path, concurrency/cancellation (if relevant).
Output format: test cases list -> patch -> test command and result.
```

## 6) API Error Contract Alignment
```text
Use CONTEXT.md + AI_RULES.md + docs/ERROR_HANDLING.md.
Task: align <handler/usecase> to go-core error contract.
Constraints: no internal detail leakage to client.
Output format: mismatch list -> patch -> sample error response.
```

## 7) Pagination Alignment
```text
Use CONTEXT.md + AI_RULES.md + docs/PAGINATION.md.
Task: implement/adjust pagination for <endpoint>.
Current strategy: <offset|cursor>.
Output format: contract summary -> query logic patch -> response example.
```

## 8) Event/Outbox Alignment
```text
Use CONTEXT.md + AI_RULES.md + docs/EVENT_CONTRACT.md + docs/TRANSACTION_RULES.md.
Task: produce event for <use case> with safe persistence.
Constraints: data change + outbox write in one transaction.
Output format: flow summary -> patch -> delivery/idempotency notes.
```
