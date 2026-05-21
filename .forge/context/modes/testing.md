---
id: mode.testing
title: Mode - Testing
type: mode
status: inferred
confidence: medium
source: ai
evidence:
  - { type: code, ref: Makefile }
owner: unresolved
updated: 2026-05-21
---

# Mode - Testing

## include

- `.forge/context/layers/testing/testing.md`
- `.forge/context/systems/go-core/system.md`

## on_demand

- `.forge/context/layers/backend/backend.md`
- `.forge/context/knowledge/assumptions.md`
- `.forge/context/knowledge/unknowns.md`
- `.forge/context/knowledge/inferred.md`

## exclude

- `.forge/context/generated/*`
- `.forge/context/temp/*`

## token_budget

6000

## notes

Concise guidance only: prefer existing Makefile and CI evidence before adding new test gates.
