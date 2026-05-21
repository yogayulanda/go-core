---
id: knowledge.adr-0001
title: ADR-0001 - Adopt Forge Context
type: knowledge
status: proposed
confidence: medium
source: ai
owner: unresolved
updated: 2026-05-21
---

# ADR-0001: Adopt Forge Context

## Status

Proposed during initialization; pending human confirmation.

## Context

The repository contains existing code, docs, and legacy AI/context artifacts. The user requested Forge v0.2.1 local context initialization without code refactor, feature implementation, behavior changes, or commit.

## Decision

Initialize `.forge` using Forge v0.2.1 standard tier with backend and testing layers and one registered library system: `go-core`.

## Consequences

- Context facts start as `inferred` or `assumption`, not `confirmed`.
- Legacy `.ai/` and root context docs remain reference material until confirmed.
- Infrastructure, frontend, and mobile layers are not active without repo evidence.
