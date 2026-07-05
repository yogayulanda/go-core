---
id: core.product
title: Product
type: core
status: inferred
confidence: high
source: ai
evidence:
  - { type: doc, ref: README.md }
  - { type: doc, ref: docs/DOMAIN.md }
  - { type: code, ref: go.mod }
owner: unresolved
updated: 2026-05-21
---

# Product

`go-core` is a reusable, domain-agnostic Go foundation library for service bootstrap and runtime concerns.

## Domain

Technical/runtime domain only: configuration, lifecycle, transports, logging, metrics, tracing, database access, migration helpers, messaging, cache, resilience, security extraction/verification, and shared error contracts.

## Users

Consuming Go services that import module `github.com/yogayulanda/go-core`.

## Scope Boundaries

- In scope: generic foundation contracts and selected platform-standard technical contracts.
- Out of scope: business entities, service-specific workflows, business states, service ownership rules, and deployment ownership.

## Canonical Source Rule

Implementation evidence wins over legacy AI/context artifacts. Existing docs are evidence when aligned with code.
