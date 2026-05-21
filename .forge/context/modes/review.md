---
id: mode.review
title: Mode - Review
type: mode
status: inferred
confidence: medium
source: ai
evidence:
  - { type: doc, ref: .forge/forge.config.yaml }
owner: unresolved
updated: 2026-05-21
---

# Mode - Review

## include

- `.forge/context/layers/backend/backend.md`
- `.forge/context/layers/testing/testing.md`
- `.forge/context/systems/go-core/system.md`
- `.forge/context/knowledge/inferred.md`
- `.forge/context/knowledge/unknowns.md`

## on_demand

- `.forge/context/knowledge/assumptions.md`
- `.forge/context/knowledge/decisions/ADR-0001-adopt-forge-context.md`

## exclude

- `.forge/context/generated/*`
- `.forge/context/temp/*`

## token_budget

6000

## notes

Concise guidance only: check evidence drift, validation attribution, regressions, and missing tests.
