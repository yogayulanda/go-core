---
id: meta.context-manifest
title: Context Manifest
type: meta
status: inferred
confidence: high
source: ai
evidence:
  - { type: doc, ref: .forge/forge.config.yaml }
owner: unresolved
updated: 2026-05-21
---

# Context Manifest

Forge runtime: `0.2.1`. Tier: `standard`.

## Bootstrap Order

1. `.forge/forge.config.yaml`
2. `.forge/context/00-meta/context-manifest.md`
3. `.forge/context/00-meta/conventions.md`
4. `.forge/context/00-meta/glossary.md`
5. `.forge/context/01-core/*`
6. `.forge/context/modes/<default_mode>.md`

## Active Layers

- `layer.backend`
- `layer.testing`

Inactive by evidence: frontend, mobile, infrastructure.

## Registered Systems

- `system.go-core` (`library`)

## File Registry

| Path | ID | Type | Status | Owner |
|---|---|---|---|---|
| `.forge/context/00-meta/context-manifest.md` | `meta.context-manifest` | meta | inferred | unresolved |
| `.forge/context/00-meta/conventions.md` | `meta.conventions` | meta | inferred | unresolved |
| `.forge/context/00-meta/glossary.md` | `meta.glossary` | meta | inferred | unresolved |
| `.forge/context/01-core/architecture.md` | `core.architecture` | core | inferred | unresolved |
| `.forge/context/01-core/constraints.md` | `core.constraints` | core | inferred | unresolved |
| `.forge/context/01-core/principles.md` | `core.principles` | core | inferred | unresolved |
| `.forge/context/01-core/product.md` | `core.product` | core | inferred | unresolved |
| `.forge/context/knowledge/assumptions.md` | `knowledge.assumptions` | knowledge | assumption | unresolved |
| `.forge/context/knowledge/confirmations.md` | `knowledge.confirmations` | knowledge | inferred | unresolved |
| `.forge/context/knowledge/decisions/ADR-0000-template.md` | `knowledge.adr-0000-template` | knowledge | unknown | unresolved |
| `.forge/context/knowledge/decisions/ADR-0001-adopt-forge-context.md` | `knowledge.adr-0001` | knowledge | proposed | unresolved |
| `.forge/context/knowledge/inferred.md` | `knowledge.inferred` | knowledge | inferred | unresolved |
| `.forge/context/knowledge/unknowns.md` | `knowledge.unknowns` | knowledge | unknown | unresolved |
| `.forge/context/layers/backend/README.md` | `layer.backend.readme` | layer | inferred | unresolved |
| `.forge/context/layers/backend/backend.md` | `layer.backend` | layer | inferred | unresolved |
| `.forge/context/layers/testing/README.md` | `layer.testing.readme` | layer | inferred | unresolved |
| `.forge/context/layers/testing/testing.md` | `layer.testing` | layer | inferred | unresolved |
| `.forge/context/modes/implementation.md` | `mode.implementation` | mode | inferred | unresolved |
| `.forge/context/modes/planning.md` | `mode.planning` | mode | inferred | unresolved |
| `.forge/context/modes/review.md` | `mode.review` | mode | inferred | unresolved |
| `.forge/context/modes/testing.md` | `mode.testing` | mode | inferred | unresolved |
| `.forge/context/systems/README.md` | `system.readme` | system | inferred | unresolved |
| `.forge/context/systems/go-core/system.md` | `system.go-core` | system | inferred | unresolved |

## Never Auto-Loaded

- `.forge/context/temp/*`
- `.forge/context/generated/*` unless explicitly requested
- Files with `status: deprecated`
