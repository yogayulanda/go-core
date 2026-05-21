---
id: mode.implementation
title: Mode - Implementation
type: mode
status: inferred
confidence: medium
source: ai
evidence:
  - { type: doc, ref: .forge/forge.config.yaml }
owner: unresolved
updated: 2026-05-21
---

# Mode - Implementation

## include

- `.forge/context/layers/backend/backend.md`
- `.forge/context/systems/go-core/system.md`
- `.forge/context/knowledge/inferred.md`

## on_demand

- `.forge/context/layers/testing/testing.md`
- `.forge/context/knowledge/assumptions.md`
- `.forge/context/knowledge/unknowns.md`
- `.forge/context/knowledge/decisions/ADR-0001-adopt-forge-context.md`

## exclude

- `.forge/context/generated/*`
- `.forge/context/temp/*`

## token_budget

8000

## notes

Concise guidance only: load active backend/system context unless the task touches tests or known gaps.
