# QUICK_PROMPTS

## How To Use
1. Pick the closest template.
2. Fill only the `<...>` placeholders.
3. Let the repo context come from `.ai/` first.
4. Add 1-2 extra constraints only when necessary.

## Global Prefix
```text
Use .ai/go-core.md + .ai/context/contracts.md + docs/ARCHITECTURE.md + docs/CODING_STANDARD.md.
Keep response token-efficient.
```

## 1) Implement Feature
```text
Use .ai/go-core.md + .ai/context/contracts.md + docs/ERROR_HANDLING.md + docs/TRANSACTION_RULES.md.
Task: <feature to build>.
Scope: <package or files>.
Constraints: foundation-oriented, explicit behavior, no hidden goroutine.
Output format: short plan -> implement patch -> test command -> risk note.
```

## 2) Code Review (Bug/Risk First)
```text
Use .ai/go-core.md + .ai/context/contracts.md + docs/CODING_STANDARD.md.
Review target: <PR/file/commit>.
Focus: bug, behavioral regression, reliability risk, missing tests.
Output format: findings by severity with file:line, then assumptions/open questions.
```

## 3) Debug Issue
```text
Use .ai/go-core.md + .ai/context/contracts.md + docs/RELIABILITY.md + docs/OBSERVABILITY.md.
Issue: <symptom or error>.
Context: <when it happens + important logs>.
Goal: find root cause and propose minimal safe fix.
Output format: hypothesis -> verification steps -> fix patch -> post-fix checks.
```

## 4) Safe Refactor
```text
Use .ai/go-core.md + .ai/context/contracts.md + docs/CODING_STANDARD.md.
Refactor target: <function/package>.
Goal: improve readability/maintainability without behavior change.
Constraints: preserve framework boundary, keep runtime wiring explicit.
Output format: plan -> patch -> equivalence notes -> tests run.
```

## 5) Add/Improve Tests
```text
Use .ai/go-core.md + docs/CODING_STANDARD.md + docs/RELIABILITY.md.
Test target: <package/function>.
Focus: edge cases, failure path, concurrency/cancellation (if relevant).
Output format: test cases list -> patch -> test command and result.
```

## 6) API Error Contract Alignment
```text
Use .ai/go-core.md + docs/ERROR_HANDLING.md.
Task: align <handler/usecase> to go-core error contract.
Constraints: no internal detail leakage to client.
Output format: mismatch list -> patch -> sample error response.
```

## 7) Pagination Alignment
```text
Use .ai/go-core.md + docs/PAGINATION.md.
Task: implement/adjust pagination for <endpoint>.
Current strategy: <offset|cursor>.
Output format: contract summary -> query logic patch -> response example.
```

## 8) Event/Outbox Alignment
```text
Use .ai/go-core.md + docs/EVENT_CONTRACT.md + docs/TRANSACTION_RULES.md.
Task: produce event for <use case> with safe persistence.
Constraints: data change + outbox write in one transaction.
Output format: flow summary -> patch -> delivery/idempotency notes.
```
