---
id: layer.backend
title: Backend Layer
type: layer
status: inferred
confidence: medium
source: ai
evidence:
  - { type: code, ref: go.mod }
  - { type: code, ref: config/validate.go }
  - { type: code, ref: app/app.go }
  - { type: code, ref: server/grpc/server.go }
  - { type: code, ref: server/gateway/gateway.go }
  - { type: code, ref: messaging/outbox/worker.go }
owner: unresolved
updated: 2026-05-21
---

# Backend Layer

## Stack

Go `1.24.3` module `github.com/yogayulanda/go-core`.

## Patterns

- Packages expose reusable runtime building blocks; consuming services register business handlers and choose optional dependencies.
- Configuration is environment-driven and validated by `config.ValidateIssues()`.
- Runtime components use explicit constructors and lifecycle registration.
- gRPC and HTTP gateway wrappers add recovery, request IDs, metrics, auth/signature handling, tracing, and error mapping.
- SQL helpers support named DB aliases, transaction propagation through context, goose migrations, and outbox SQL for `mysql`, `postgres`, and `sqlserver`.

## Validation and Fallback Attribution

Use `core.constraints` as the canonical source for validation-layer attribution, DB constraint separation, and repository/runtime fallback semantics.

## Non-Source Areas

`.forge/context/generated/`, `.forge/context/temp/`, examples, and templates are references or generated/scratch support, not implementation source of truth.
